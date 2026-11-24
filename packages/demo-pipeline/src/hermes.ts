import type { Article, Job, ResearchBrief, Source, Story } from '@creata/contracts'
import { clone, fail, find, list, localized, pick, requireRole, validUrl, type Body, type Context, type DemoState, type DemoStory } from './model'
import { createArticle, publishArticle, remember, storyPermalinkConflicts, storyPermalinkMessage } from './editorial'
import { isFollowableStory } from './story-content'

const phases=['classification','research','draft','review'] as const
const active=(j:Job)=>['queued','running'].includes(j.status)
export function enqueue(c:Context,kind:Job['kind'],payload:Body):Job {
 const existing=c.state.jobs.find(j=>j.kind===kind&&active(j)&&((payload.articleId&&j.payload.articleId===payload.articleId)||(payload.sourceId&&j.payload.sourceId===payload.sourceId)||(!payload.sourceId&&payload.url&&j.payload.url===payload.url)))
 if(existing)return existing
 const job:Job={id:c.id('job'),kind,status:'queued',payload:{...clone(payload),simulated:true},attempts:1,createdAt:c.iso,updatedAt:c.iso,progress:{phase:kind==='draft'||kind==='rework'?'classification':kind,status:'queued',updatedAt:c.iso},checkpoints:{}}
 c.state.jobs.unshift(job);c.audit('job_queued',job.id,{kind,simulated:true});return job
}
function researchFor(article:Article,story?:Story):ResearchBrief {const refs=article.sources;return {facts:refs.map(source=>({claim:story?.text?.slice(0,200)||article.lead.ru||'Демонстрационный материал: факт требует редакционной проверки.',sourceUrls:[source.url],numbers:[],dates:[]})),conflicts:[],singleSourceClaims:refs.length<2?['Демонстрация: материал основан на одном источнике. Проверьте его перед реальной публикацией.']:[],retrievedEvidence:story?.evidence?.map(e=>clone(e))??refs.map(source=>({...source,text:article.lead.ru||'Локально подготовленная демонстрационная выдержка; внешний сайт не запрашивался.'})),unavailableSources:[]}}
function finishDraft(c:Context,j:Job) {
 const article=c.state.articles.find(a=>a.id===j.payload.articleId);if(!article){j.status='failed';j.error='Материал для задания отсутствует';return}
 if(j.payload.baseRevision!==undefined&&article.revision!==j.payload.baseRevision){j.status='failed';j.error='Материал изменён после запуска Hermes. Обновления сохранены; повторный запуск требует новой редакции.';return}
 const story=c.state.stories.find(s=>s.id===j.payload.storyId);const sample=c.state.articles.find(a=>a.status==='published'&&a.categoryId===article.categoryId)??Object.values(c.state.publications)[0]
 const base=sample??article;const note=String(j.payload.note??'');const simulated=localized('Демонстрационный черновик Hermes. Тексты и исследования смоделированы локально; проверьте факты в первоисточниках.','Hermes демо қолжазбасы. Мәтіндер мен зерттеу жергілікті үлгіленген; деректерді түпнұсқадан тексеріңіз.','Hermes demo draft. Text and research are simulated locally; verify every claim against the original sources.')
 if(j.kind==='rework')article.body={ru:`${article.body.ru}\n\nРедакционная доработка (демо): ${note}\n${simulated.ru}`,kk:`${article.body.kk}\n\nРедакторлық түзету (демо): ${note}\n${simulated.kk}`,en:`${article.body.en}\n\nEditorial revision (demo): ${note}\n${simulated.en}`}
 else {article.lead=clone(base.lead);article.body={ru:`${base.body.ru}\n\n${simulated.ru}`,kk:`${base.body.kk||base.lead.kk}\n\n${simulated.kk}`,en:`${base.body.en||base.lead.en}\n\n${simulated.en}`};article.summary=clone(base.summary);article.actions=localized('Изучите первоисточник, проверьте условия и сохраните связанные возможности.','Түпнұсқаны қарап, шарттарын тексеріп, байланысты мүмкіндіктерді сақтаңыз.','Read the original source, check eligibility and save the related opportunities.')}
 if(!article.sources.length)article.sources=clone(story?.sources?.length?story.sources:base.sources)
 if(!article.media)article.media=clone(base.media);article.author=c.state.authors.find(a=>a.id===article.authorId)??c.state.authors[0];article.researchBrief=researchFor(article,story);article.opportunityIds=c.state.opportunities.filter(o=>o.status==='approved'&&(!o.categoryId||o.categoryId===article.categoryId)).map(o=>o.id);article.reviewNotes=[...article.reviewNotes,'Демо Hermes: перед реальной публикацией требуется проверка редактора.'];article.status='review';article.revision++;article.approvedRevision=undefined
 if(j.payload.graphic||article.format==='graphic'){article.format='graphic';article.graphic={type:'bar',unit:'demo index',period:'2024–2026 · simulated',sourceUrl:article.sources[0]?.url||'https://example.org',sourceName:'Demo · unverified',methodology:localized('Данные сгенерированы для демонстрации интерфейса; они не отражают реальную статистику.','Көрсеткіштер интерфейс демонстрациясы үшін жасалған.','Values are generated for the demo interface and do not represent real statistics.'),verified:false,labels:['2024','2025','2026'].map(x=>localized(x)),series:[{name:localized('Демонстрационный ряд','Демо қатар','Demo series'),values:[24,42,61]}]}}
 remember(c,article,note);j.result={articleId:article.id,article:clone(article),simulated:true,costUsd:0.0042};j.checkpoints??={};j.checkpoints.research={status:'succeeded',output:clone(article.researchBrief) as unknown as Body,completedAt:c.iso};j.checkpoints.draft={status:'succeeded',output:{articleId:article.id,locales:['ru','kk','en'],simulated:true},completedAt:c.iso};j.checkpoints.review={status:'succeeded',output:{reviewNotes:article.reviewNotes,humanApprovalRequired:true,simulated:true},completedAt:c.iso};c.state.settings.ai.spentTodayUsd=Number((c.state.settings.ai.spentTodayUsd+0.0042).toFixed(6))
}
function finishCollect(c:Context,j:Job) {
 const source=c.state.sources.find(s=>s.id===j.payload.sourceId);const url=String(j.payload.url??source?.url??'');if(!url){j.status='failed';j.error='Источник удалён. Создайте новое задание.';return}
 const existing=c.state.stories.find(s=>s.url===url&&s.status!=='merged');if(existing){j.result={storyIds:[existing.id],collected:0,existing:true,simulated:true};return}
 const sample=Object.values(c.state.publications).find(a=>a.categoryId===source?.topics[0])??Object.values(c.state.publications)[0];const topic=String((j.payload.topics as string[]|undefined)?.[0]??source?.topics[0]??sample?.categoryId??c.state.categories[0]?.id??'ai');const title=sample?`${sample.title.ru} · демо-сюжет`:`Демо-сюжет: ${new URL(url).hostname}`
 const story:Story={id:`demo-collected-${j.id}`,title,text:`Демонстрационный сбор по ссылке ${url}. Внешний сайт не запрашивался.\n\n${sample?.lead.ru??'Редактор проверит первоисточник, контекст и пользу материала для аудитории.'}`,url,sourceId:source?.id??'manual',sourceType:source?.type??'manual',locale:(j.payload.locale as Story['locale'])??source?.locale??'ru',categoryId:topic,region:sample?.region??'KZ',tags:clone(sample?.tags??['demo']),score:87,reason:'Демо-оценка 87/100: релевантная рубрика, источник и практическая польза. Оценка смоделирована.',status:'new',sources:[{url,title:source?.name??new URL(url).hostname}],createdAt:c.iso,evidence:[{url,title:source?.name??title,text:'Локально подготовленная демо-выдержка. Реальный сбор и проверка источника не выполнялись.',sourceId:source?.id}]};c.state.stories.unshift(story);j.result={storyIds:[story.id],collected:1,simulated:true};j.checkpoints={classification:{status:'succeeded',output:{score:story.score,categoryId:topic,reason:story.reason,simulated:true},completedAt:c.iso}}
}
/** Project elapsed jobs from persisted inputs. Readers never write or emit events. */
export function advanceJobs(c:Context) {
 for(const j of [...c.state.jobs].reverse()) {
  if(!active(j))continue
  const elapsed=Math.max(0,c.time-Date.parse(j.updatedAt));const start=Date.parse(j.updatedAt)
  if(elapsed<1000)continue
  j.status='running';const isDraft=['draft','rework'].includes(j.kind);const phase=isDraft?phases[Math.min(3,Math.floor(elapsed/2500))]:j.kind;j.progress={phase,status:'running',updatedAt:new Date(Math.min(c.time,start+10000)).toISOString()};j.checkpoints??={}
  if(isDraft){for(let index=0;index<4;index++){if(elapsed<index*2500)continue;const name=phases[index];j.checkpoints[name]={status:elapsed>=(index+1)*2500?'succeeded':'running',output:{simulated:true,...(name==='classification'?{score:87,reason:'Демонстрационная классификация',categoryId:c.state.articles.find(a=>a.id===j.payload.articleId)?.categoryId}:{})},usage:[{reservationId:`demo:${j.id}:${name}`,costUsd:0.00105,inputTokens:900,outputTokens:240}],updatedAt:new Date(start+index*2500).toISOString()}}}
  if(elapsed<10000)continue
  const completed=new Date(start+10000).toISOString();const completedContext={...c,iso:completed};j.status='succeeded'
  if(isDraft)finishDraft(completedContext,j);else if(j.kind==='collect')finishCollect(completedContext,j);else j.result={simulated:true,reachable:true,message:'Демонстрационная проверка: формат ссылки принят. Сеть не использовалась.'}
  j.progress={phase:isDraft?'review':j.kind,status:j.status,updatedAt:completed};j.updatedAt=completed
  const source=c.state.sources.find(s=>s.id===j.payload.sourceId);if(source){source.lastRunAt=completed;source.nextRunAt=new Date(start+10000+source.intervalMinutes*60000).toISOString();source.lastError=j.error}
 }
 for(const a of c.state.articles){
  if(a.scheduledAt&&Date.parse(a.scheduledAt)<=c.time&&a.status==='approved'&&a.approvedRevision===a.revision){
   const scheduled={...c,iso:a.scheduledAt}
   if(storyPermalinkConflicts(c,a)){
    a.status='review'
    a.approvedRevision=undefined
    a.scheduledAt=undefined
    a.reviewNotes=[...a.reviewNotes,storyPermalinkMessage]
    remember(scheduled,a)
    continue
   }
   if(a.storyUpdates?.length)a.updatedAt=scheduled.iso
   publishArticle(scheduled,a)
   if(isFollowableStory(a))remember(scheduled,a)
  }
 }
}
function sourceValid(body:Body) {if(!String(body.name??'').trim())fail(400,'Укажите название источника');validUrl(body.url);if(!['rss','website','telegram'].includes(body.type))fail(400,'Неизвестный тип источника');if(body.type==='telegram'&&!['t.me','telegram.me'].includes(new URL(body.url).hostname))fail(400,'Используйте публичную ссылку t.me');if(!Number.isInteger(body.intervalMinutes)||body.intervalMinutes<1||body.intervalMinutes>1440)fail(400,'Интервал должен быть от 1 до 1440 минут')}
const storyVersion=(story:DemoStory)=>story.updatedAt??story.createdAt
function touchStory(c:Context,story:DemoStory) {story.updatedAt=new Date(Math.max(c.time,Date.parse(storyVersion(story))+1)).toISOString()}
function splitEvidence<T extends {url:string}>(left:T[],right:T[],beforeLeft:T[],beforeRight:T[]) {
 const originalLeft=new Set(beforeLeft.map(item=>item.url)),originalRight=new Set(beforeRight.map(item=>item.url));const kept=left.filter(item=>!originalRight.has(item.url)||originalLeft.has(item.url)),donor=left.filter(item=>originalRight.has(item.url)&&!originalLeft.has(item.url));return {left:kept,right:[...right,...donor.filter(item=>!right.some(x=>x.url===item.url))]}
}
function evidence(s:DemoState,id:string) {const story=find(s.stories,id);return {story,rawItems:(story.evidence??[]).map((e,i)=>({id:`${id}:raw:${i}`,...e})),linkedArticleIds:s.jobs.filter(j=>j.payload.storyId===id).map(j=>String(j.payload.articleId)).filter(Boolean),version:storyVersion(story)}}
export function hermes(c:Context,method:string,p:string[],body:Body):unknown|undefined {
 if(p[0]!=='admin')return;const s=c.state
 if(p[1]==='sources') {
  if(method==='GET'&&p.length===2)return list(s.sources)
  if(method==='POST'&&p.length===2){requireRole(s,['admin','chief_editor','editor']);const source:Source={id:c.id('source'),name:'',url:'',type:'rss',intervalMinutes:30,locale:'ru',topics:[],active:true,...pick(body,['name','type','url','locale','topics','intervalMinutes','active'])};sourceValid(source);s.sources.push(source);c.audit('source_created',source.id);return source}
  if(p[2]){const source=find(s.sources,p[2]);if(method==='PATCH'&&p.length===3){requireRole(s,['admin','chief_editor','editor']);const updated={...source,...pick(body,['name','type','url','locale','topics','intervalMinutes','active'])};sourceValid(updated);Object.assign(source,updated);c.audit('source_updated',source.id);return source}if(method==='DELETE'&&p.length===3){requireRole(s,['admin','chief_editor']);s.sources=s.sources.filter(x=>x.id!==source.id);s.deleted.push(`sources:${source.id}`);c.audit('source_deleted',source.id);return {deleted:true}}if(method==='POST'&&['run','check','test'].includes(p[3])){requireRole(s,['admin','chief_editor','editor']);return enqueue(c,p[3]==='run'?'collect':'check',{sourceId:source.id,source:clone(source),locale:source.locale,topics:source.topics})}}
 }
 if(['jobs','runs'].includes(p[1])) {
  if(method==='GET'&&p.length===2)return list(p[1]==='runs'?s.jobs.filter(j=>['check','collect'].includes(j.kind)):s.jobs)
  if(p[2]){const j=find(s.jobs,p[2]);if(method==='GET'&&p.length===3)return j;if(method==='POST'&&p[3]==='retry'){requireRole(s,['admin','chief_editor','editor']);if(['queued','running','succeeded'].includes(j.status))return j;j.status='queued';j.attempts++;j.error=undefined;j.updatedAt=c.iso;j.progress={phase:j.kind,status:'queued',updatedAt:c.iso};j.checkpoints={};c.audit('job_retried',j.id);return j}if(method==='POST'&&p[3]==='cancel'){requireRole(s,['admin','chief_editor','editor']);if(active(j)){j.status='cancelled';j.updatedAt=c.iso;j.progress={phase:j.kind,status:'cancelled',updatedAt:c.iso};c.audit('job_cancelled',j.id)}return j}if(method==='POST'&&p[3]==='fail'){requireRole(s,['admin','chief_editor']);if(active(j)){j.status='failed';j.error=String(body.error||'Демонстрационная ошибка, созданная редактором');j.updatedAt=c.iso;c.audit('job_demo_failed',j.id)}return j}}
 }
 if(p[1]!=='bank')return
 if(method==='GET'&&p.length===2)return list(s.stories.filter(story=>story.status!=='merged'))
 if(method==='POST'&&p[2]==='collect'){requireRole(s,['admin','chief_editor','editor','author']);const url=validUrl(body.url);const existing=s.jobs.find(j=>j.kind==='collect'&&j.payload.url===url&&j.status!=='cancelled'&&j.status!=='failed');return existing??enqueue(c,'collect',{url,locale:['ru','kk','en'].includes(body.locale)?body.locale:'ru',topics:Array.isArray(body.topics)?body.topics:[]})}
 if(method==='GET'&&p[2]==='similar'){const items=[];const stories=s.stories.filter(s=>s.status!=='merged');for(let i=0;i<stories.length;i++)for(let j=i+1;j<stories.length;j++){const l=stories[i],r=stories[j];if(l.categoryId===r.categoryId||l.sources.some(x=>r.sources.some(y=>x.url===y.url)))items.push({leftId:l.id,rightId:r.id,leftTitle:l.title,rightTitle:r.title,leftUpdatedAt:evidence(s,l.id).version,rightUpdatedAt:evidence(s,r.id).version,score:l.url===r.url?0.94:0.76,reasons:['Демо: совпадает рубрика или ссылка на источник']})}return list(items)}
 if(p[2]==='merges'){
  if(method==='GET'&&p.length===3)return list(s.merges)
  if(method==='POST'&&['undo','split'].includes(p[4])){
   requireRole(s,['admin','chief_editor','editor']);const merge=find(s.merges,p[3]);const left=find(s.stories,merge.leftId),right=find(s.stories,merge.rightId)
   if(merge.status!=='merged'||left.status==='merged'||left.mergedInto||right.status!=='merged'||right.mergedInto&&right.mergedInto!==left.id)fail(409,'Объединение изменилось. Обновите сравнение.')
   if(p[4]==='split'&&(body.expectedLeftVersion!==storyVersion(left)||body.expectedRightVersion!==storyVersion(right)))fail(409,'Сюжеты изменились. Обновите сравнение.')
   if(p[4]==='undo'&&(merge.afterLeftVersion&&merge.afterLeftVersion!==storyVersion(left)||merge.afterRightVersion&&merge.afterRightVersion!==storyVersion(right)))fail(409,'После объединения сюжеты изменились. Используйте разделение с сохранением изменений.')
   // Return donor-only evidence to the donor while retaining evidence added later.
   const sources=splitEvidence(left.sources,right.sources,merge.leftBefore.sources,merge.rightBefore.sources);left.sources=sources.left;right.sources=sources.right
   const proofs=splitEvidence(left.evidence??[],right.evidence??[],merge.leftBefore.evidence??[],merge.rightBefore.evidence??[]);left.evidence=proofs.left;right.evidence=proofs.right
   left.mergedStoryIds=(left.mergedStoryIds??[]).filter(id=>id!==right.id&&!(merge.rightBefore.mergedStoryIds??[]).includes(id));delete right.mergedInto;right.status=merge.rightBefore.status;touchStory(c,left);touchStory(c,right);merge.status=p[4]==='split'?'split':'undone';c.audit(`stories_${merge.status}`,merge.id);return merge
  }
 }
 if(!p[2])return
 const story=find(s.stories,p[2]);if(method==='GET'&&p[3]==='evidence')return evidence(s,story.id)
 if(method==='POST'&&['draft','graphic'].includes(p[3])) {requireRole(s,['chief_editor','editor','author','admin']);if(story.status==='merged'||story.mergedInto)fail(409,'Сюжет объединён. Откройте основной сюжет.');const existing=s.jobs.find(j=>j.payload.storyId===story.id&&Boolean(j.payload.graphic)===(p[3]==='graphic'));if(existing&&s.articles.some(a=>a.id===existing.payload.articleId))return find(s.articles,String(existing.payload.articleId));const sample=Object.values(s.publications).find(a=>a.categoryId===story.categoryId)??Object.values(s.publications)[0];const a=createArticle(c,{title:sample?.title??localized(story.title,'Демо сюжет: '+story.title,'Demo story: '+story.title),categoryId:story.categoryId,region:story.region,tags:story.tags,sources:story.sources,media:sample?.media,format:p[3]==='graphic'?'graphic':'article'});a.status='writing';enqueue(c,'draft',{articleId:a.id,baseRevision:a.revision,storyId:story.id,story:clone(story),graphic:p[3]==='graphic'});story.status='drafted';touchStory(c,story);return a}
 if(method==='POST'&&p[3]==='merge'){
  requireRole(s,['admin','chief_editor','editor']);const other=find(s.stories,body.otherId);if(other.id===story.id)fail(400,'Выберите два разных сюжета');if(story.status==='merged'||other.status==='merged'||story.mergedInto||other.mergedInto||body.expectedUpdatedAt!==storyVersion(story)||body.otherExpectedUpdatedAt!==storyVersion(other))fail(409,'Сюжеты изменились. Обновите сравнение.')
  const merge={id:c.id('merge'),leftId:story.id,rightId:other.id,status:'merged',createdAt:c.iso,leftBefore:clone(story),rightBefore:clone(other),afterLeftVersion:'',afterRightVersion:''};s.merges.unshift(merge);story.sources=[...story.sources,...other.sources.filter(x=>!story.sources.some(y=>y.url===x.url))];story.evidence=[...(story.evidence??[]),...(other.evidence??[])];story.mergedStoryIds=[...new Set([...(story.mergedStoryIds??[]),other.id,...(other.mergedStoryIds??[])])];other.status='merged';other.mergedInto=story.id;touchStory(c,story);touchStory(c,other);merge.afterLeftVersion=storyVersion(story);merge.afterRightVersion=storyVersion(other);c.audit('stories_merged',merge.id);return merge
 }
}
