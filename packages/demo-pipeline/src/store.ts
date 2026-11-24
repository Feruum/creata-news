import type { Article, Author, NewspaperIssue, NewspaperPublication, Story, User } from '@creata/contracts'
import { clone, DEMO_PIPELINE_EVENT, DEMO_PIPELINE_KEY, fail, localized, type DemoArticle, type DemoClock, type DemoSeed, type DemoState, type DemoStorage } from './model'
import { decodeState, encodeState } from './codec'
import { isFollowableStory, storyPublication } from './story-content'

const restrictedBrowserMemory=new Map<string,string>()
function defaultStorage():DemoStorage {if(typeof window==='undefined')return new Map();try{return window.localStorage}catch{return restrictedBrowserMemory}}
function makeUser(id:string,name:string,roles=['chief_editor','admin']):User {return {id,name,roles,locale:'ru',stage:'building',topics:[],frequency:'daily',paused:false,verticals:[],telegramLinked:false}}
export function seedState(seed:DemoSeed,time:number):DemoState {
 const iso=new Date(time).toISOString();const authors=clone(seed.authors??[]);for(const a of seed.articles){if(a.author&&!authors.some(x=>x.id===a.author!.id))authors.push(clone(a.author))}
 if(!authors.length)authors.push({id:seed.articles[0]?.authorId||'demo-author',name:'Редактор демоверсии',bio:localized('Демонстрационная редакция Creata News','Creata News демо редакциясы','Creata News demo desk'),role:'Demo chief editor'})
 const actor=authors[0];const user=makeUser(actor.id,actor.name);user.verticals=seed.categories.map(c=>c.id)
 const articles:DemoArticle[]=clone(seed.articles).map(a=>({...a,author:a.author??authors.find(x=>x.id===a.authorId),...(a.status==='published'?{publishedRevision:a.publishedRevision??a.revision,approvedRevision:a.approvedRevision??a.revision}:{})}))
 for(const article of articles)if(isFollowableStory(article))article.storySlug=article.slug
 const publications=Object.fromEntries(articles.filter(a=>a.status==='published').map(a=>[a.id,storyPublication(a)]))
 const sources:DemoState['sources']=[{id:'demo-source-research',name:'Исследования и технологии · demo',type:'rss',url:'https://news.mit.edu/rss/topic/artificial-intelligence2',locale:'en',topics:[seed.categories[0]?.id||'ai'],intervalMinutes:60,active:true},{id:'demo-source-central-asia',name:'Центральная Азия · demo',type:'website',url:'https://astanahub.com/',locale:'ru',topics:[seed.categories[1]?.id||'startups'],intervalMinutes:120,active:true},{id:'demo-source-telegram',name:'Creata News · demo',type:'telegram',url:'https://t.me/creata_news',locale:'ru',topics:[],intervalMinutes:30,active:false}]
 const stories:Story[]=articles.slice(0,4).map((a,i)=>({id:`demo-story-${i+1}`,title:a.title.ru,text:a.lead.ru+'\n\n'+a.body.ru,url:a.sources[0]?.url||sources[0].url,sourceId:sources[i%2].id,sourceType:sources[i%2].type,locale:'ru',categoryId:a.categoryId,region:a.region,tags:clone(a.tags),score:92-i*7,reason:'Демонстрационная оценка: подтверждённый источник, новизна, польза читателю.',status:'new',sources:clone(a.sources),createdAt:iso,evidence:a.sources.map(s=>({...s,text:a.lead.ru,sourceId:sources[i%2].id}))}))
 if(stories.length===1)stories.push({...clone(stories[0]),id:'demo-story-related',title:stories[0].title+' · связанный источник',score:79})
 const original=articles[0];if(original){for(const [i,status] of ['review','writing','approved'].entries()){const draft:DemoArticle={...clone(original),id:`demo-editorial-${status}`,slug:`demo-editorial-${status}`,title:{ru:`[Демо] ${original.title.ru}`,kk:`[Демо] ${original.title.kk}`,en:`[Demo] ${original.title.en}`},status:status as Article['status'],revision:1,publishedRevision:undefined,publishedAt:undefined,approvedRevision:status==='approved'?1:undefined,authorId:user.id,author:clone(actor),createdAt:iso,updatedAt:iso,reviewNotes:['Демонстрационный материал для проверки редакционного процесса.'],format:i===1?'graphic':original.format};delete draft.storySlug;if(i===1)draft.graphic={type:'bar',unit:'demo index',period:'2024–2026 · simulated',sourceUrl:original.sources[0]?.url||'https://example.org',sourceName:'Demo source',methodology:localized('Демонстрационные данные. Не использовать для выводов.'),verified:false,labels:['2024','2025','2026'].map(x=>localized(x)),series:[{name:localized('Демо'),values:[24,40,58]}]};articles.push(draft)}}
 const newspapers:NewspaperIssue[]=[];const newspaperPublications:Record<string,NewspaperPublication>={};for(const item of seed.newspapers??[]){const issue:NewspaperIssue={...clone(item),items:'items'in item?clone(item.items):item.entries.map(e=>({id:e.id,articleId:e.articleId,photo:e.photo})),status:'published',createdAt:iso,updatedAt:iso,creatorId:user.id};newspapers.push(issue);if('entries'in item)newspaperPublications[issue.id]=clone(item)}
 const pages=clone(seed.pages??[]);const state:DemoState={version:1,articles,publications,categories:clone(seed.categories),authors,opportunities:clone(seed.opportunities),launches:clone(seed.launches??[]),pages,pagePublications:Object.fromEntries(pages.filter(p=>p.status==='published').map(p=>[p.slug,clone(p)])),pageRevisions:Object.fromEntries(pages.map(p=>[p.slug,[clone(p)]])),newspapers,newspaperPublications,media:[],sources,stories,jobs:[],settings:{editorialInstructions:'Демо: проверяйте первоисточники, отделяйте факты от предположений и объясняйте пользу читателю. AI, исследования и бюджет здесь моделируются локально.',scoreThreshold:65,ai:{endpoint:'https://demo.invalid/local-simulation',model:'creata-demo-multilingual',keyConfigured:false,dailyBudgetUsd:10,spentTodayUsd:0.012,inputPricePerMillion:0,outputPricePerMillion:0},digestHour:9,timezone:'Asia/Qyzylorda'},session:{user,csrfToken:'demo-local-session',providers:{google:false,telegram:false},devAuth:true},users:[user,makeUser('demo-reader','Демо подписчик',['reader']),makeUser('demo-editor','Демо редактор',['editor'])],digests:[],deliveries:[],audit:[{id:'demo-audit-initial',actorId:user.id,action:'demo_initialized',targetId:'browser-local',createdAt:iso,detail:'Simulation only; no AI or Telegram service is called.'}],revisions:Object.fromEntries(articles.map(a=>[a.id,[clone(a)]])),merges:[],deleted:[],sequence:100,saved:{articles:[],opportunities:[]},followedStories:{},seenStoryUpdateIds:{},storyFollowUps:{}}
 const urls=new Set<string>();for(const a of articles){if(a.media?.url&&!urls.has(a.media.url)){urls.add(a.media.url);state.media.push({...clone(a.media),id:`demo-media-${state.media.length+1}`,revision:1,needsRightsReview:true,referenced:true,references:[{kind:'article',id:a.id}]})}}
 state.jobs=[{id:'demo-job-completed',kind:'check',status:'succeeded',payload:{sourceId:sources[0].id,source:clone(sources[0]),simulated:true},result:{simulated:true,message:'Демонстрационная проверка завершена; сеть не использовалась.'},attempts:1,createdAt:iso,updatedAt:iso},{id:'demo-job-failed',kind:'collect',status:'failed',payload:{sourceId:sources[1].id,source:clone(sources[1]),simulated:true},error:'Демонстрационная ошибка доступа. Нажмите «Повторить», чтобы пройти восстановление.',attempts:1,createdAt:iso,updatedAt:iso},{id:'demo-job-queued',kind:'check',status:'queued',payload:{sourceId:sources[2].id,source:clone(sources[2]),simulated:true},attempts:0,createdAt:iso,updatedAt:iso}]
 return state
}
function mergeMissing(current:any,defaults:any):any {
 if(current===undefined)return clone(defaults)
 if(!current||!defaults||typeof current!=='object'||typeof defaults!=='object'||Array.isArray(current)||Array.isArray(defaults))return current
 for(const [key,value]of Object.entries(defaults))current[key]=mergeMissing(current[key],value)
 return current
}
export function createStore(seed:DemoSeed,storage:DemoStorage=defaultStorage(),now:DemoClock=Date.now) {
 const clock=()=>Number(now());const initial=seedState(seed,clock());let fallback:string|undefined;let mode:'unprobed'|'persistent'|'memory'='unprobed'
 const readRaw=()=>{
  if(mode==='memory')return fallback
  try{const raw=storage instanceof Map?storage.get(DEMO_PIPELINE_KEY):storage.getItem(DEMO_PIPELINE_KEY);mode='persistent';return raw}
  catch{if(mode==='unprobed'){mode='memory';return fallback}fail(503,'Не удалось прочитать сохранённую демоверсию. Повторите действие, когда хранилище браузера станет доступно.','demo_storage_unavailable')}
 }
 const read=():DemoState=>{
  const raw=readRaw();if(!raw)return clone(initial)
  try{
   const parsed=decodeState(raw);if(parsed.version!==1||!Array.isArray(parsed.articles))return clone(initial)
   // A missing key in an existing publication map means an explicit unpublish.
   // Never merge seed snapshots back into these maps after that action.
   const snapshots=Object.fromEntries(['publications','pagePublications','newspaperPublications'].filter(key=>parsed[key]!==undefined).map(key=>[key,clone(parsed[key])]))
   const existingIds=new Set<string>(parsed.articles.map((a:Article)=>a.id))
   const merged=mergeMissing(parsed,clone(initial)) as DemoState;Object.assign(merged,snapshots)
   for(const key of ['articles','categories','authors','opportunities','launches','pages','newspapers','sources','stories','media'] as const){
    const items=merged[key] as {id:string}[]
    for(const item of initial[key]){
     const current=items.find(x=>x.id===item.id)
     if(current){
      const defaults=clone(item) as Record<string,unknown>
      for(const field of ['approvedRevision','publishedRevision','publishedAt','scheduledAt','lastError'])delete defaults[field]
      // Omitted story fields on a saved draft may have been explicitly cleared.
      if(key==='articles'){delete defaults.actionGuide;delete defaults.storyUpdates;delete defaults.storySlug}
      mergeMissing(current,defaults)
     }
     else if(!merged.deleted.includes(`${key}:${item.id}`))items.push(clone(item))
    }
   }
   for(const [id,article]of Object.entries(initial.publications))if(!existingIds.has(id))merged.publications[id]=clone(article)
   // Recover the editorial permalink lock for demo records saved before it was stored.
   for(const article of merged.articles){
    if(article.storySlug)continue
    const published=merged.publications[article.id]
    const story=published&&isFollowableStory(published)?published:merged.revisions[article.id]?.find(isFollowableStory)
    if(story)article.storySlug=story.slug
   }
   return merged
  }catch{return clone(initial)}
 }
 const write=(state:DemoState)=>{
  const raw=encodeState(state);if(mode==='unprobed')readRaw()
  if(mode==='memory')fallback=raw
  else{
   try{if(storage instanceof Map)storage.set(DEMO_PIPELINE_KEY,raw);else storage.setItem(DEMO_PIPELINE_KEY,raw)}
   catch(error){if((error as {name?:string})?.name==='QuotaExceededError')fail(507,'Хранилище демоверсии заполнено. Изменение не сохранено. Удалите неиспользуемые изображения или загрузите файл меньшего размера.','demo_storage_full');fail(503,'Не удалось сохранить демоверсию в браузере. Изменение не сохранено; повторите действие.','demo_storage_unavailable')}
   fallback=raw
  }
  if(typeof window!=='undefined'&&typeof window.CustomEvent==='function')window.dispatchEvent(new CustomEvent(DEMO_PIPELINE_EVENT,{detail:{key:DEMO_PIPELINE_KEY}}))
 }
 return {read,write,clock,reset:()=>write(clone(initial))}
}
