import type { Article, Digest } from '@creata/contracts'
import { canPublish, clone, expected, fail, find, list, localized, pick, requireRole, type Body, type Context, type DemoArticle } from './model'
import { enqueue } from './hermes'
import { applyStoryContent, isFollowableStory, requireCompleteStoryContent, storyPublication } from './story-content'
import { storyFollowUp, storyFollowUpQueue } from './story-follow-up'

const contentFields=['slug','categoryId','region','authorId','editorId','title','lead','body','summary','actions','layer','tags','sources','reviewNotes','media','format','graphic','opportunityIds']
export function createArticle(c:Context,body:Body):Article {
 const user=c.state.session.user!;const id=c.id('article');const empty=localized();const a:Article={id,slug:id,categoryId:c.state.categories[0]?.id||'ai',region:'KZ',authorId:user.id,author:c.state.authors.find(a=>a.id===user.id)??c.state.authors[0],layer:'world',title:empty,lead:empty,body:empty,summary:empty,actions:empty,tags:[],sources:[],reviewNotes:[],...pick(body,contentFields),status:'new',revision:1,createdAt:c.iso,updatedAt:c.iso}
 if(typeof a.title==='string')a.title=localized(a.title);if(!a.title||!['ru','kk','en'].some(l=>a.title[l as keyof typeof a.title]?.trim()))fail(400,'Укажите заголовок материала','title_required')
 applyStoryContent(a,body)
 a.author=c.state.authors.find(x=>x.id===a.authorId)??a.author;c.state.articles.unshift(a);c.state.revisions[a.id]=[clone(a)];c.audit('article_created',a.id);return a
}
export function canEdit(c:Context,a:Article) {const u=c.state.session.user!;return u.roles.includes('chief_editor')||u.roles.includes('editor')&&u.verticals.includes(a.categoryId)||u.roles.includes('author')&&a.authorId===u.id}
export function allowedActions(c:Context,a:Article) {
 const u=c.state.session.user!
 if(canPublish(u,a))return ['submit','rework','reject','approve','publish','schedule','unpublish']
 if(u.roles.includes('editor')&&u.verticals.includes(a.categoryId))return ['submit','rework']
 return u.roles.includes('author')&&a.authorId===u.id?['submit']:[]
}
export function remember(c:Context,a:DemoArticle,note?:string) {a.updatedAt=c.iso;(c.state.revisions[a.id]??=[]).unshift({...clone(a),...(note?{note}:{})} as DemoArticle)}
export const storyPermalinkMessage='Адрес опубликованного сюжета нельзя изменить'
export function storyPermalinkConflicts(c:Context,article:DemoArticle) {
 const permalink=article.storySlug||(article.storyUpdates?.length?c.state.publications[article.id]?.slug:undefined)
 return !!permalink&&article.slug!==permalink
}
function requireStoryPermalink(c:Context,article:DemoArticle) {
 if(storyPermalinkConflicts(c,article))fail(409,storyPermalinkMessage,'story_slug_locked')
}
export function publishArticle(c:Context,article:DemoArticle) {
 article.status='published'
 article.publishedRevision=article.revision
 article.publishedAt=c.iso
 article.scheduledAt=undefined
 if(isFollowableStory(article))article.storySlug??=article.slug
 c.state.publications[article.id]=storyPublication(article)
}
function publicArticles(c:Context,q:URLSearchParams) {let items=Object.values(c.state.publications);for(const field of ['categoryId','layer','region','format'] as const){const value=q.get(field);if(value)items=items.filter(a=>a[field]===value)}const search=q.get('q')?.toLowerCase();if(search)items=items.filter(a=>Object.values(a.title).some(t=>t.toLowerCase().includes(search)));return list(items.sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||'')))}
function digest(c:Context,body:Body):Digest {const locale=['ru','kk','en'].includes(body.locale)?body.locale:'ru';const articles=Object.values(c.state.publications).filter(a=>a.title[locale as 'ru']);const opportunities=c.state.opportunities.filter(o=>o.status==='approved');return {id:c.id('digest'),locale,kind:body.kind||'channel',status:'draft',articles:clone(articles),opportunities:clone(opportunities),createdAt:c.iso,text:`[ДЕМО · SIMULATION · Ничего не отправляется]\n\n${articles.map(a=>a.title[locale as 'ru']+'\n'+a.summary[locale as 'ru']).join('\n\n')}\n\n${opportunities.map(o=>o.title[locale as 'ru']+' — '+o.url).join('\n')}`}}
export function editorial(c:Context,method:string,p:string[],body:Body,q:URLSearchParams):unknown|undefined {
 const s=c.state;const path=p.join('/')
 if(method==='GET'&&path==='public/categories')return list(s.categories)
 if(method==='GET'&&['public/articles','public/popular'].includes(path))return publicArticles(c,q)
 if(method==='GET'&&p[0]==='public'&&p[1]==='articles'&&p[2])return Object.values(s.publications).find(a=>a.id===p[2]||a.slug===p[2])??fail(404,'Опубликованный материал не найден')
 if(method==='GET'&&path==='public/authors')return list(s.authors)
 if(method==='GET'&&path==='public/opportunities')return list(s.opportunities.filter(o=>o.status==='approved'))
 if(p[0]!=='admin')return
 if(path==='admin/story-follow-ups')return storyFollowUpQueue(c)
 if(p[1]==='articles') {
  if(p.length===2){if(method==='GET')return list(s.articles.filter(a=>(!q.get('format')||a.format===q.get('format'))&&(!q.get('status')||a.status===q.get('status'))));if(method==='POST'){requireRole(s,['chief_editor','editor','author']);return createArticle(c,body)}return}
  const a=find(s.articles,p[2]);if(method==='GET'&&p.length===3)return a
  if(p[3]==='follow-up')return storyFollowUp(c,a,method,body)
  if(method==='GET'&&p[3]==='revisions')return list(s.revisions[a.id]??[a])
  if(method==='GET'&&p[3]==='research')return list(s.jobs.filter(j=>j.payload.articleId===a.id).sort((a,b)=>b.createdAt.localeCompare(a.createdAt)))
  if(method==='PATCH'&&p.length===3){
   if(!canEdit(c,a))fail(403,'Редактирование доступно автору, редактору рубрики или главному редактору')
   expected(a,body)
   applyStoryContent(a,body)
   Object.assign(a,pick(body,contentFields))
   requireStoryPermalink(c,a)
   Object.assign(a,{revision:a.revision+1,status:'writing',approvedRevision:undefined,scheduledAt:undefined})
   a.author=s.authors.find(x=>x.id===a.authorId)??a.author
   remember(c,a)
   c.audit('article_updated',a.id)
   return a
  }
  if(method==='POST'&&p[3]==='graphic-draft'){if(!canEdit(c,a))fail(403,'Недостаточно прав');expected(a,body);return enqueue(c,'draft',{articleId:a.id,baseRevision:a.revision,graphic:true,note:body.note||'Демонстрационный график'})}
  if(method==='POST'&&p[3]==='actions') {
   if(!allowedActions(c,a).includes(body.action))fail(403,'Действие недоступно для этой роли');expected(a,body)
   if(body.action==='submit'){a.status='review';a.approvedRevision=undefined;a.scheduledAt=undefined}
   else if(body.action==='approve'){
    if(!['new','writing','review','rework','published','unpublished'].includes(a.status))fail(409,'Материал нельзя одобрить на текущем этапе')
    requireStoryPermalink(c,a)
    requireCompleteStoryContent(a)
    a.status='approved';a.approvedRevision=a.revision;a.scheduledAt=undefined
   }
   else if(body.action==='publish'||body.action==='schedule'){
    if(!['approved','scheduled'].includes(a.status)||a.approvedRevision!==a.revision)fail(409,'Сначала одобрите текущую редакцию материала','approval_required')
    requireStoryPermalink(c,a)
    requireCompleteStoryContent(a)
    if(body.action==='schedule'){const at=Date.parse(body.scheduledAt);if(!Number.isFinite(at)||at<=c.time)fail(400,'Укажите дату публикации в будущем');a.status='scheduled';a.scheduledAt=new Date(at).toISOString();remember(c,a,body.note);c.audit('article_scheduled',a.id,{revision:a.revision,scheduledAt:a.scheduledAt});return a}
    if(a.publishedRevision&&a.publishedRevision!==a.revision)a.corrections=[...(a.corrections??[]),{revision:a.revision,publishedAt:c.iso,note:body.note||'Демонстрационная редакционная правка'}]
    publishArticle(c,a)
   }else if(body.action==='unpublish'){if(!s.publications[a.id]&&a.status!=='unpublished')fail(409,'Снять с публикации можно только опубликованный материал');delete s.publications[a.id];a.status='unpublished';a.publishedRevision=undefined;a.approvedRevision=undefined;a.scheduledAt=undefined}
   else if(body.action==='reject'||body.action==='rework'){if(!String(body.note??'').trim())fail(400,'Укажите комментарий редактора');a.status=body.action==='reject'?'rejected':'rework';a.approvedRevision=undefined;a.scheduledAt=undefined;a.reviewNotes=[...a.reviewNotes,String(body.note)];if(body.action==='rework')enqueue(c,'rework',{articleId:a.id,baseRevision:a.revision,note:String(body.note)})}
   else fail(400,'Неизвестное действие')
   remember(c,a,body.note);c.audit(`article_${body.action}`,a.id,{revision:a.revision,note:body.note});return a
  }
 }
 if(p[1]==='digests') {
  if(method==='GET'&&p.length===2)return list(s.digests)
  if(method==='POST'&&(p[2]==='preview'||p.length===2)){requireRole(s,['chief_editor','editor','admin']);const d=digest(c,body);s.digests.unshift(d);c.audit('digest_created',d.id);return d}
  if(p[2]&&!['preview'].includes(p[2])){const d=find(s.digests,p[2]);if(method==='GET'&&p.length===3)return d;if(method==='POST'&&p[3]==='approve'){requireRole(s,['chief_editor','admin']);if(!['draft','approved'].includes(d.status))fail(409,'Этот дайджест уже обработан');d.status='approved';c.audit('digest_approved',d.id);return d}if(method==='POST'&&p[3]==='send'){requireRole(s,['chief_editor','admin']);if(d.status==='sent')return d;if(d.status!=='approved')fail(409,'Сначала одобрите дайджест');d.status='sent';s.deliveries.unshift({id:c.id('delivery'),digestId:d.id,chatId:'demo:channel:creata-news',status:'sent',attempts:1,createdAt:c.iso,simulated:true,messageId:s.sequence});c.audit('digest_simulated_send',d.id,{simulated:true,externalMessages:0});return d}}
 }
 if(p[1]==='deliveries'){if(method==='GET'&&p.length===2)return list(s.deliveries);if(method==='POST'&&p[3]==='resolve'){requireRole(s,['chief_editor','admin']);const d=find(s.deliveries,p[2]);if(!['retry','mark_sent'].includes(body.action))fail(400,'Неизвестное действие');if(body.action==='mark_sent'&&(!Number.isInteger(body.messageId)||body.messageId<1))fail(400,'Укажите положительный идентификатор сообщения');if(d.status!=='sent'){d.status='sent';d.attempts++;d.messageId=body.messageId||s.sequence;d.error=undefined;c.audit('delivery_simulated_resolve',d.id,{simulated:true})}return d}}
}
