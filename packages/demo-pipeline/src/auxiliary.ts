import type { Author, Launch, MediaAsset, NewspaperIssue, NewspaperPublication, Opportunity, SitePage, User } from '@creata/contracts'
import { clone, expected, fail, find, list, localized, pick, requireRole, validUrl, type Body, type Context } from './model'
import { followedStoryRequest } from './story-content'

const roles=['reader','author','editor','chief_editor','admin']
export function auth(c:Context,method:string,p:string[],body:Body):unknown|undefined {
 const s=c.state
 if(p[0]==='auth'){
  if(method==='GET'&&p[1]==='me')return s.session
  if(method==='POST'&&p[1]==='logout'){s.session.user=null;s.session.csrfToken=undefined;c.audit('demo_logout','session');return {ok:true}}
  if(method==='POST'&&p[1]==='dev-login'){const role=String(body.role||'chief_editor');if(!roles.includes(role))fail(400,'Неизвестная роль демоверсии');const author=s.authors[0];const user:User={id:author.id,name:author.name,roles:role==='chief_editor'?['chief_editor','admin']:[role],locale:'ru',stage:'building',topics:[],frequency:'daily',paused:false,verticals:s.categories.map(x=>x.id),telegramLinked:false};s.session={user,csrfToken:'demo-local-session',providers:{google:false,telegram:false},devAuth:true};const index=s.users.findIndex(u=>u.id===user.id);if(index>=0)s.users[index]=clone(user);else s.users.push(clone(user));c.audit('demo_role_changed',user.id,{role});return s.session}
  if(p[1]==='google'||p[1]==='telegram')fail(501,'В демоверсии нет внешней авторизации. Используйте локальный вход с ролью.','demo_external_unavailable')
 }
 if(p[0]==='account'){
  if(!s.session.user)fail(401,'Войдите в демоверсию');const user=s.session.user
  if(method==='PATCH'&&p.length===1){Object.assign(user,pick(body,['name','locale','stage','topics','frequency','paused']));return user}
  if(p[1]==='saved'){if(method==='GET')return {articles:s.articles.filter(a=>s.saved.articles.includes(a.id)).map(a=>s.publications[a.id]).filter(Boolean),opportunities:s.opportunities.filter(o=>s.saved.opportunities.includes(o.id))};if(['POST','DELETE'].includes(method)&&['articles','opportunities','article','opportunity'].includes(p[2])){const kind=p[2].startsWith('article')?'articles':'opportunities';if(method==='POST'&&!s.saved[kind].includes(p[3]))s.saved[kind].push(p[3]);if(method==='DELETE')s.saved[kind]=s.saved[kind].filter(id=>id!==p[3]);return {ok:true}}}
  if(p[1]==='followed-stories'){
   const following={
    followedStoryIds:Object.hasOwn(s.followedStories,user.id)?s.followedStories[user.id]:[],
    seenStoryUpdateIds:Object.hasOwn(s.seenStoryUpdateIds,user.id)?s.seenStoryUpdateIds[user.id]:{},
   }
   const result=followedStoryRequest(Object.values(s.publications),following,method,p.slice(2),body)
   if(method!=='GET'){
    s.followedStories={...s.followedStories,[user.id]:following.followedStoryIds}
    s.seenStoryUpdateIds={...s.seenStoryUpdateIds,[user.id]:following.seenStoryUpdateIds}
   }
   return result
  }
  if(p[1]==='telegram-link')fail(501,'Связка Telegram недоступна в локальной демоверсии','demo_external_unavailable')
 }
}
function publication(c:Context,issue:NewspaperIssue):NewspaperPublication {
 const {items,...rest}=issue;const entries=items.map(item=>{const article=c.state.publications[item.articleId];if(!article)fail(409,'В выпуск можно включить только опубликованные материалы');return {id:item.id,articleId:item.articleId,article:clone(article),photo:clone(item.photo??article.media??{url:'',caption:'',credit:'',rights:'Demo only'})}})
 return {...clone(rest),publishedRevision:issue.revision,pages:entries.length+1,entries,format:'managed'}
}
function referenced(c:Context,asset:MediaAsset) {const references:Array<{kind:string;id:string}>=[];for(const a of c.state.articles)if(a.media?.url===asset.url)references.push({kind:'article',id:a.id});for(const n of c.state.newspapers)if(n.cover.url===asset.url||n.items.some(i=>i.photo?.url===asset.url))references.push({kind:'newspaper',id:n.id});return {...asset,referenced:!!references.length,references}}
type PreparedUpload=Pick<MediaAsset,'url'|'credit'|'rights'|'caption'|'mime'|'bytes'>
/** Finish asynchronous file IO before a mutation captures browser state. */
export async function prepareMediaUpload(form:FormData):Promise<PreparedUpload> {
 const file=form.get('file');if(!(file instanceof Blob)||!file.size)fail(400,'Выберите изображение');if(file.size>2*1024*1024)fail(413,'Демо хранит изображения до 2 МБ в этом браузере. Выберите файл меньшего размера.','demo_upload_limit');if(!['image/jpeg','image/png','image/webp','image/gif','image/avif'].includes(file.type))fail(415,'Демо поддерживает JPEG, PNG, WebP, GIF и AVIF')
 const rights=String(form.get('rights')??'').trim(),credit=String(form.get('credit')??'').trim(),caption=String(form.get('caption')??'');if(!credit||!rights)fail(400,'Укажите автора и права использования')
 const bytes=new Uint8Array(await file.arrayBuffer());let binary='';for(let i=0;i<bytes.length;i+=8192)binary+=String.fromCharCode(...bytes.subarray(i,i+8192));return {url:`data:${file.type};base64,${btoa(binary)}`,credit,rights:`${rights} · Demo browser-local upload; verify rights before production.`,caption,mime:file.type,bytes:file.size}
}
export function uploadMedia(c:Context,prepared:PreparedUpload) {
 requireRole(c.state,['chief_editor','admin','editor','author']);const asset:MediaAsset={...prepared,id:c.id('media'),revision:1,createdAt:c.iso,needsRightsReview:true,referenced:false,references:[]};c.state.media.unshift(asset);c.audit('media_demo_uploaded',asset.id,{bytes:prepared.bytes,simulated:true});return asset
}
export function auxiliary(c:Context,method:string,p:string[],body:Body,q:URLSearchParams):unknown|undefined {
 const s=c.state;const path=p.join('/')
 if(p[0]==='public'&&p[1]==='launches'){const items=s.launches.filter(l=>l.status==='published');if(method==='GET'&&p.length===2)return list(items);if(method==='GET'&&p.length===3)return items.find(l=>l.slug===p[2]||l.id===p[2])??fail(404,'Запуск не найден')}
 if(p[0]==='public'&&p[1]==='pages'&&method==='GET')return s.pagePublications[p[2]]??fail(404,'Страница не опубликована')
 if(p[0]==='public'&&p[1]==='newspapers'&&method==='GET'){const items=Object.values(s.newspaperPublications);return p[2]?items.find(n=>n.slug===p[2]||n.id===p[2])??fail(404,'Выпуск не опубликован'):list(items)}
 if(p[0]!=='admin')return
 if(path==='admin/settings'){if(method==='GET')return s.settings;if(method==='PATCH'){requireRole(s,['chief_editor','admin']);const fields=pick(body,['editorialInstructions','scoreThreshold','digestHour','timezone']);if(fields.scoreThreshold!==undefined&&(!Number.isFinite(fields.scoreThreshold)||fields.scoreThreshold<0||fields.scoreThreshold>100))fail(400,'Порог должен быть от 0 до 100');Object.assign(s.settings,fields);if(body.ai){const limits=pick(body.ai,['dailyBudgetUsd','inputPricePerMillion','outputPricePerMillion']);if(Object.values(limits).some(x=>!Number.isFinite(x)||x<0))fail(400,'Демо-бюджет должен быть положительным');Object.assign(s.settings.ai,limits,{endpoint:'https://demo.invalid/local-simulation',model:'creata-demo-multilingual',keyConfigured:false})}c.audit('demo_settings_updated','settings');return s.settings}}
 if(path==='admin/audit'&&method==='GET'){
  requireRole(s,['admin','chief_editor'])
  return list(s.audit)
 }
 if(path==='admin/staff'&&method==='GET')return list(s.users.filter(u=>u.roles.some(r=>r!=='reader')).map(({id,name,roles,verticals})=>({id,name,roles,verticals})))
 if(p[1]==='users'){if(method==='GET'&&p.length===2)return list(s.users);if(method==='PATCH'&&p.length===3){requireRole(s,['admin','chief_editor']);const user=find(s.users,p[2]);if(body.roles&&(!Array.isArray(body.roles)||body.roles.some((r:string)=>!roles.includes(r))))fail(400,'Неизвестная роль');Object.assign(user,pick(body,['roles','verticals']));if(s.session.user?.id===user.id)s.session.user=clone(user);c.audit('demo_user_updated',user.id);return user}}
 if(path==='admin/analytics'&&method==='GET'){
  const statuses=Object.fromEntries(['new','writing','review','rework','approved','published','rejected'].map(status=>[status,s.articles.filter(a=>a.status===status).length]));const stages=Object.keys(statuses);return {published:Object.keys(s.publications).length,stories:s.stories.filter(x=>x.status!=='merged').length,subscribers:s.users.filter(u=>u.roles.includes('reader')).length,views:1240,saves:67+s.saved.articles.length,opportunityClicks:43,statuses,aiCostUsd:s.settings.ai.spentTodayUsd,simulated:true,daily:Array.from({length:7},(_,i)=>({date:new Date(c.time-(6-i)*86400000).toISOString().slice(0,10),views:[132,164,141,188,172,201,242][i]})),stageMetrics:{articleCount:s.articles.length,durations:stages.map(stage=>({stage,completedVisits:s.audit.filter(a=>a.action===`article_${stage==='review'?'submit':stage==='approved'?'approve':stage==='published'?'publish':stage}`).length,averageSeconds:120,medianSeconds:100,p90Seconds:240,currentCount:statuses[stage],averageCurrentAgeSeconds:Math.max(0,s.articles.filter(a=>a.status===stage).reduce((sum,a)=>sum+(c.time-Date.parse(a.updatedAt))/1000,0)/(statuses[stage]||1))})),funnel:stages.map(stage=>({stage,count:statuses[stage],rate:statuses[stage]/Math.max(1,s.articles.length)})),transitions:[]}}
 }
 if(['authors','opportunities','launches'].includes(p[1])) {
  const kind=p[1] as 'authors'|'opportunities'|'launches';if(method==='GET'&&p.length===2)return list<Author|Opportunity|Launch>(s[kind])
  if(method==='POST'&&p.length===2){requireRole(s,['chief_editor','admin','editor']);const id=c.id(kind);let item:Author|Opportunity|Launch
   if(kind==='opportunities'&&body.needsRightsReview!==undefined&&typeof body.needsRightsReview!=='boolean')fail(400,'Отметка проверки прав должна быть логическим значением')
   if(kind==='authors'){if(!String(body.name??'').trim())fail(400,'Укажите имя автора');item={id,name:body.name,bio:localized(),...pick(body,['name','bio','avatarUrl','role'])} as Author}
   else if(kind==='opportunities'){validUrl(body.url);item={id,title:localized(),description:localized(),organization:'',type:'grant',url:body.url,country:'KZ',deadline:'',status:'pending',...pick(body,['title','description','organization','type','url','country','deadline','status','categoryId','photo','forWhom','photoRights','languages','needsRightsReview'])} as Opportunity}
   else{validUrl(body.url);if(s.launches.some(l=>l.slug===body.slug))fail(409,'Этот адрес запуска уже занят');item={id,slug:id,name:'',tagline:localized(),description:localized(),team:localized(),url:body.url,launchDate:c.iso.slice(0,10),country:'KZ',categoryId:s.categories[0]?.id??'ai',status:'draft',...pick(body,['slug','name','tagline','description','team','url','launchDate','country','categoryId','photo','status','logoUrl','tags']),revision:1,createdAt:c.iso,updatedAt:c.iso} as Launch}
   ;(s[kind] as Array<Author|Opportunity|Launch>).unshift(item);c.audit(`${kind}_created`,id);return item
  }
  if(method==='PATCH'&&p.length===3){requireRole(s,['chief_editor','admin','editor']);const item=find(s[kind] as Array<Author|Opportunity|Launch>,p[2]);if(kind==='launches')expected(item as Launch,body);if(kind==='opportunities'&&body.needsRightsReview!==undefined&&typeof body.needsRightsReview!=='boolean')fail(400,'Отметка проверки прав должна быть логическим значением');if(body.url)validUrl(body.url);const fields=kind==='authors'?['name','bio','avatarUrl','role']:kind==='opportunities'?['title','description','organization','type','url','country','deadline','status','categoryId','photo','forWhom','photoRights','languages','needsRightsReview']:['slug','name','tagline','description','team','url','launchDate','country','categoryId','photo','status','logoUrl','tags'];Object.assign(item,pick(body,fields));if(kind==='launches'){(item as Launch).revision++;(item as Launch).updatedAt=c.iso}c.audit(`${kind}_updated`,item.id);return item}
  if(method==='DELETE'&&p.length===3){requireRole(s,['chief_editor','admin']);find<Author|Opportunity|Launch>(s[kind],p[2]);if(kind==='authors')s.authors=s.authors.filter(x=>x.id!==p[2]);else if(kind==='opportunities')s.opportunities=s.opportunities.filter(x=>x.id!==p[2]);else s.launches=s.launches.filter(x=>x.id!==p[2]);s.deleted.push(`${kind}:${p[2]}`);c.audit(`${kind}_deleted`,p[2]);return {deleted:true}}
 }
 if(p[1]==='pages') {
  if(method==='GET'&&p.length===2)return list(s.pages)
  const page=s.pages.find(page=>page.slug===p[2]||page.id===p[2]);if(!page)fail(404,'Страница не найдена в данных демоверсии')
  if(method==='GET'&&p.length===3)return page;if(method==='GET'&&p[3]==='revisions')return list(s.pageRevisions[page.slug]??[page])
  if(method==='PATCH'&&p.length===3){requireRole(s,['chief_editor','admin','editor']);expected(page,body);if(!body.content||typeof body.content!=='object')fail(400,'Нужно содержимое страницы');page.content=clone(body.content);page.revision++;page.status='draft';page.updatedAt=c.iso;(s.pageRevisions[page.slug]??=[]).unshift(clone(page));c.audit('page_updated',page.id);return page}
  if(method==='POST'&&['publish','unpublish'].includes(p[3])){requireRole(s,['chief_editor','admin']);expected(page,body);page.status=p[3]==='publish'?'published':'draft';if(p[3]==='publish'){page.publishedRevision=page.revision;s.pagePublications[page.slug]=clone(page)}else{delete s.pagePublications[page.slug];page.publishedRevision=undefined}c.audit(`page_${p[3]}`,page.id);return page}
 }
 if(p[1]==='media') {
  if(method==='GET'&&p.length===2){const search=q.get('q')?.toLowerCase()??'';return {...list(s.media.map(m=>referenced(c,m)).filter(m=>[m.caption,m.credit,m.rights,m.url].join(' ').toLowerCase().includes(search))),acceptedTypes:['image/jpeg','image/png','image/webp','image/gif','image/avif']}}
  if(method==='POST'&&p[2]==='cleanup'){requireRole(s,['chief_editor','admin']);const unused=s.media.filter(m=>!referenced(c,m).referenced);for(const m of unused)s.deleted.push(`media:${m.id}`);s.media=s.media.filter(m=>referenced(c,m).referenced);c.audit('media_cleanup','media',{removed:unused.length});return {removed:unused.length,remaining:s.media.length}}
  if(p[2]){const m=find(s.media,p[2]);if(method==='PATCH'&&p.length===3){requireRole(s,['chief_editor','admin','editor','author']);expected(m,body);Object.assign(m,pick(body,['caption','credit','rights']));m.revision++;if(body.confirmRights)m.needsRightsReview=false;c.audit('media_updated',m.id);return referenced(c,m)}if(method==='DELETE'&&p.length===3){requireRole(s,['chief_editor','admin','editor']);expected(m,body);if(referenced(c,m).referenced)fail(409,'Изображение используется в материале или выпуске');s.media=s.media.filter(x=>x.id!==m.id);s.deleted.push(`media:${m.id}`);c.audit('media_deleted',m.id);return {deleted:true}}}
 }
 if(p[1]==='newspapers') {
  if(method==='GET'&&p.length===2)return list(s.newspapers)
  if(method==='POST'&&p.length===2){requireRole(s,['chief_editor','admin','editor','author']);if(!body.title||!body.slug)fail(400,'Укажите название и адрес выпуска');if(s.newspapers.some(n=>n.slug===body.slug))fail(409,'Адрес выпуска уже занят');const issue:NewspaperIssue={id:c.id('newspaper'),slug:body.slug,number:String(body.number??''),date:body.date??c.iso.slice(0,7),title:body.title,description:body.description??'',locale:body.locale??'ru',cover:body.cover??{url:'',caption:'',credit:'',rights:''},items:clone(body.items??[]),status:'draft',revision:1,creatorId:s.session.user!.id,createdAt:c.iso,updatedAt:c.iso};s.newspapers.unshift(issue);c.audit('newspaper_created',issue.id);return issue}
  if(p[2]){const issue=find(s.newspapers,p[2]);if(method==='GET'&&p.length===3)return issue;if(method==='GET'&&p[3]==='preview')return publication(c,issue);if(method==='PATCH'&&p.length===3){requireRole(s,['chief_editor','admin','editor','author']);expected(issue,body);Object.assign(issue,pick(body,['slug','number','date','title','description','locale','cover','items']),{revision:issue.revision+1,updatedAt:c.iso,status:'draft'});c.audit('newspaper_updated',issue.id);return issue}if(method==='POST'&&['publish','unpublish'].includes(p[3])){requireRole(s,['chief_editor','admin']);expected(issue,body);if(p[3]==='publish'){if(!issue.items.length)fail(400,'Добавьте опубликованные материалы в выпуск');const snapshot=publication(c,issue);issue.status='published';issue.publishedAt=c.iso;issue.publishedRevision=issue.revision;s.newspaperPublications[issue.id]={...snapshot,publishedAt:c.iso}}else{issue.status='draft';issue.publishedRevision=undefined;issue.publishedAt=undefined;delete s.newspaperPublications[issue.id]}c.audit(`newspaper_${p[3]}`,issue.id);return issue}}
 }
}
