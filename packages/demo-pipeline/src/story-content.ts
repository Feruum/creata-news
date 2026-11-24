import type { Article, FollowedStories, LocalizedText } from '@creata/contracts'
import { clone, fail, validUrl, type Body, type DemoArticle, type FollowedStoryState } from './model'

type StoryUpdate = NonNullable<Article['storyUpdates']>[number]
const locales = ['ru','kk','en'] as const
const storyUpdateId = /^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$/

function fields(value:unknown, names:readonly string[]) {
 if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(key=>!names.includes(key))) {
  fail(400,'Неверный формат данных сюжета')
 }
 return value as Record<string,unknown>
}

function storyText(value:unknown, max:number):LocalizedText {
 const text=fields(value,locales)
 const {ru,kk,en}=text
 if(typeof ru!=='string'||typeof kk!=='string'||typeof en!=='string'||[ru,kk,en].some(item=>item.includes('\0')||[...item].length>max)) {
  fail(400,'Проверьте длину и переводы текста сюжета')
 }
 return {ru,kk,en}
}

function storySource(value:unknown) {
 if(typeof value!=='string'||[...value].length>2048)fail(400,'Укажите корректную ссылку на источник сюжета')
 const source=value.trim()
 if(source){
  const authority=/^https?:\/\/([^/?#]+)/i.exec(source)?.[1]
  if(!authority||authority.includes('@')||/[\u0000-\u001f\u007f\\]/.test(source))fail(400,'Укажите корректную ссылку на источник сюжета')
  validUrl(source)
 }
 return value
}

function validUpdateDate(value:unknown):value is string {
 if(typeof value!=='string')return false
 const date=/^(\d{4})-(\d{2})-(\d{2})T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d{1,9})?(?:Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$/.exec(value)
 if(!date||!Number.isFinite(Date.parse(value)))return false
 const calendar=new Date(`${date[1]}-${date[2]}-01T00:00:00Z`)
 calendar.setUTCMonth(calendar.getUTCMonth()+1,0)
 return Number(date[3])>0&&Number(date[3])<=calendar.getUTCDate()
}

function readStoryUpdates(value:unknown):StoryUpdate[] {
 if(!Array.isArray(value)||value.length>40)fail(400,'Добавьте не более 40 обновлений сюжета')
 const ids=new Set<string>()
 return value.map(entry=>{
  const {id,occurredAt,title,body,sourceUrl,articleSlug}=fields(entry,['id','occurredAt','title','body','sourceUrl','articleSlug'])
  if(typeof id!=='string'||!storyUpdateId.test(id)||ids.has(id))fail(400,'Укажите уникальный идентификатор обновления')
  if(!validUpdateDate(occurredAt))fail(400,'Укажите дату и часовой пояс обновления')
  if(articleSlug!==undefined&&(typeof articleSlug!=='string'||articleSlug==='.'||articleSlug==='..'||
   articleSlug.trim()!==articleSlug||!/^[^\p{Cc}/\\?#]{1,120}$/u.test(articleSlug))) {
   fail(400,'Укажите корректный адрес связанного материала')
  }
  ids.add(id)
  return {id,occurredAt,title:storyText(title,300),body:storyText(body,4000),sourceUrl:storySource(sourceUrl),...(articleSlug===undefined?{}:{articleSlug})}
 })
}

export function applyStoryContent(article:Article,body:Body) {
 if(body.actionGuide!==undefined){
  const guide=body.actionGuide===null?undefined:fields(body.actionGuide,['audience','steps','sourceUrl'])
  if(guide)article.actionGuide={audience:storyText(guide.audience,300),steps:storyText(guide.steps,4000),sourceUrl:storySource(guide.sourceUrl)}
  else delete article.actionGuide
 }
 if(body.storyUpdates!==undefined)article.storyUpdates=readStoryUpdates(body.storyUpdates)
}

export function requireCompleteStoryContent(article:Article) {
 const hasPair=(left:LocalizedText,right:LocalizedText)=>locales.some(locale=>left[locale].trim()&&right[locale].trim())
 const guide=article.actionGuide
 if(guide&&(!guide.sourceUrl.trim()||!hasPair(guide.audience,guide.steps))) {
  fail(400,'Заполните аудиторию, шаги и источник инструкции')
 }
 if(article.storyUpdates?.some(update=>!update.sourceUrl.trim()||!hasPair(update.title,update.body))) {
  fail(400,'Заполните заголовок, текст и источник каждого обновления')
 }
}

export function storyPublication(article:DemoArticle):Article {
 const snapshot=clone(article)
 snapshot.reviewNotes=[]
 delete snapshot.researchBrief
 delete snapshot.storySlug
 const publicText=({ru,kk,en}:LocalizedText):LocalizedText=>({ru,kk,en})
 const guide=snapshot.actionGuide
 if(guide)snapshot.actionGuide={audience:publicText(guide.audience),steps:publicText(guide.steps),sourceUrl:guide.sourceUrl.trim()}
 if(snapshot.storyUpdates?.length) {
  snapshot.storyUpdates=snapshot.storyUpdates.map(({id,occurredAt,title,body,sourceUrl,articleSlug})=>({
   id,occurredAt,title:publicText(title),body:publicText(body),sourceUrl:sourceUrl.trim(),...(articleSlug?{articleSlug}:{}),
  })).sort((a,b)=>compareUpdateDates(b.occurredAt,a.occurredAt))
 }else delete snapshot.storyUpdates
 return snapshot
}

function compareUpdateDates(a:string,b:string):number {
 const subMilliseconds=(date:string)=>Number(/\.(\d{1,9})/.exec(date)?.[1].padEnd(9,'0').slice(3)??0)
 return Date.parse(a)-Date.parse(b)||subMilliseconds(a)-subMilliseconds(b)
}

export function isFollowableStory<T extends Article>(article:T):article is T & {storyUpdates:[StoryUpdate,...StoryUpdate[]]} {
 return article.status==='published'&&!!article.storyUpdates?.length
}

export function listFollowedStories(articles:Article[], ids:string[], seenUpdateIds:Record<string,string[]>={}):FollowedStories {
 const followed=new Set(ids)
 const items=articles.filter(isFollowableStory).filter(article=>followed.has(article.id)).map(article=>({
  article,
  latest:article.storyUpdates.reduce((latest,{occurredAt})=>compareUpdateDates(occurredAt,latest)>0?occurredAt:latest,article.storyUpdates[0].occurredAt),
 }))
 items.sort((a,b)=>compareUpdateDates(b.latest,a.latest)||a.article.id.localeCompare(b.article.id))
 const unreadUpdateIds=Object.fromEntries(items.map(({article})=>{
  const seen=new Set(Object.hasOwn(seenUpdateIds,article.id)?seenUpdateIds[article.id]:[])
  return [article.id,article.storyUpdates.map(({id})=>id).filter(id=>!seen.has(id))]
 }))
 return {items:items.map(({article})=>article),unreadUpdateIds}
}

export function followedStoryRequest(articles:Article[],state:FollowedStoryState,method:string,parts:string[],body:Body) {
 const {followedStoryIds,seenStoryUpdateIds}=state
 if(method==='GET'&&parts.length===0)return listFollowedStories(articles,followedStoryIds,seenStoryUpdateIds)

 const [articleId,action]=parts
 if(method==='DELETE'&&parts.length===1){
  state.followedStoryIds=followedStoryIds.filter(id=>id!==articleId)
  delete seenStoryUpdateIds[articleId]
  return {ok:true}
 }
 const markRead=parts.length===2&&action==='read'
 if(method!=='POST'||(parts.length!==1&&!markRead))fail(404,'Запрос для сюжета не найден')

 let submitted:Set<string>|undefined
 if(markRead){
  const updateIds:unknown=body?.updateIds
  if(!Array.isArray(updateIds)||updateIds.length>40||Object.keys(body).length!==1||updateIds.some(id=>typeof id!=='string'||!storyUpdateId.test(id))){
   fail(400,'Укажите корректные идентификаторы обновлений')
  }
  submitted=new Set<string>(updateIds)
 }
 const article=articles.find(item=>item.id===articleId)
 if(!article||!isFollowableStory(article))fail(404,'Опубликованный сюжет не найден')
 const following=followedStoryIds.includes(articleId)
 if(submitted){
  if(!following)fail(404,'Подписка на сюжет не найдена')
  const current=Object.hasOwn(seenStoryUpdateIds,articleId)?seenStoryUpdateIds[articleId]:[]
  const read=article.storyUpdates.map(({id})=>id).filter(id=>submitted.has(id))
  state.seenStoryUpdateIds={...seenStoryUpdateIds,[articleId]:[...new Set([...current,...read])]}
 }else if(!following){
  state.followedStoryIds=[...followedStoryIds,articleId]
  state.seenStoryUpdateIds={...seenStoryUpdateIds,[articleId]:article.storyUpdates.map(({id})=>id)}
 }
 return {ok:true}
}
