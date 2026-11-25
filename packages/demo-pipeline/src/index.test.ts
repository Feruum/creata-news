import { describe, expect, test } from 'bun:test'
import { ApiError, type Article, type Digest, type FollowedStories, type Job, type Source, type Story } from '../../contracts/src/index'
import { createPipelineDemo, DEMO_PIPELINE_KEY } from './index'

const text = (ru: string) => ({ru, kk: `ҚАЗ: ${ru}`, en: `EN: ${ru}`})
const fixture: Article = { id:'published', slug:'published-story', title:text('Опубликованный материал'), lead:text('Лид'), body:text('Текст на основе источника'), summary:text('Кратко'), actions:text('Изучить источник'), categoryId:'ai', region:'KZ', authorId:'demo-author', layer:'world', sources:[{url:'https://example.org/report',title:'Report'}], reviewNotes:[],tags:['AI'],media:{url:'/demo-media/photo.webp',caption:'Фото',credit:'Demo',rights:'Demo only'},status:'published',revision:1,createdAt:'2026-11-15T00:00:00Z',updatedAt:'2026-11-15T00:00:00Z',publishedAt:'2026-11-15T00:00:00Z'}
const seed = { articles:[fixture], categories:[{id:'ai',slug:'ai',name:text('AI')}], opportunities:[{id:'op',title:text('AI fellowship'),description:text('For builders'),organization:'Demo lab',type:'grant',url:'https://example.org/grant',country:'KZ',deadline:'2027-12-31',status:'approved' as const,categoryId:'ai'}] }
const input=(method:string,body?:unknown):RequestInit=>({method,...(body===undefined?{}:{body:JSON.stringify(body)})})
function setup(storage=new Map<string,string>()) { let time=Date.parse('2026-11-15T12:00:00Z');const demo=createPipelineDemo(seed,storage,()=>time);return {demo,storage,advance:(ms=20000)=>{time+=ms}} }
const guide={audience:text('Предприниматели'),steps:text('Проверьте условия приема'),sourceUrl:'https://example.org/guide'}
const storyUpdate={id:'call-opens',occurredAt:'2026-11-20T16:00:00Z',title:text('Открыт прием заявок'),body:text('Заявки принимают до конца месяца'),sourceUrl:'https://example.org/announcement'}

describe('browser-local pipeline',()=>{
test('a story can link to a published article whose slug contains punctuation',async()=>{
 const articleSlug='ручная-проверка-·-акселератор-«орбита»-bbc572df'
 const demo=createPipelineDemo({...seed,articles:[fixture,{...fixture,id:'related',slug:articleSlug}]},new Map())
 const draft=await demo.request<Article>('/admin/articles/published',input('PATCH',{
  expectedRevision:1,storyUpdates:[{...storyUpdate,articleSlug}],
 }))
 expect(draft.storyUpdates?.[0]?.articleSlug).toBe(articleSlug)
 for(const invalid of ['.','..','article/other','article\\other','article#heading','article?query','article\0']) {
  await expect(demo.request('/admin/articles/published',input('PATCH',{
   expectedRevision:draft.revision,storyUpdates:[{...storyUpdate,articleSlug:invalid}],
  }))).rejects.toMatchObject({status:400})
 }
})
test('story fields remain private while editing and can be cleared in a later publication',async()=>{
 const storySeed={...seed,articles:[{...fixture,actionGuide:guide,storyUpdates:[storyUpdate]}]}
 const storage=new Map<string,string>()
 const demo=createPipelineDemo(storySeed,storage)
 const correction={...storyUpdate,body:text('Продлили прием заявок')}
 const draft=await demo.request<Article>('/admin/articles/published',input('PATCH',{expectedRevision:1,actionGuide:null,storyUpdates:[correction]}))
 expect(draft.actionGuide).toBeUndefined()
 expect(draft.storyUpdates).toEqual([correction])
 const revisions=await demo.request<{items:Article[]}>('/admin/articles/published/revisions')
 expect(revisions.items[0]?.storyUpdates).toEqual([correction])
 expect(demo.getArticle('published-story')?.actionGuide).toEqual(guide)
 expect(demo.getArticle('published-story')?.storyUpdates).toEqual([storyUpdate])
 await demo.request('/admin/articles/published/actions',input('POST',{action:'approve',expectedRevision:2}))
 await demo.request('/admin/articles/published/actions',input('POST',{action:'publish',expectedRevision:2}))
 expect(demo.getArticle('published-story')?.storyUpdates).toEqual([correction])
 const cleared=await demo.request<Article>('/admin/articles/published',input('PATCH',{expectedRevision:2,storyUpdates:[]}))
 expect(cleared.storyUpdates).toEqual([])
 await demo.request('/admin/articles/published/actions',input('POST',{action:'approve',expectedRevision:3}))
 await demo.request('/admin/articles/published/actions',input('POST',{action:'publish',expectedRevision:3}))
 const restarted=createPipelineDemo(storySeed,storage)
 expect(restarted.getArticle('published-story')?.actionGuide).toBeUndefined()
 expect(restarted.getArticle('published-story')?.storyUpdates).toBeUndefined()
 const saved=await restarted.request<Article>('/admin/articles/published')
 expect(saved.actionGuide).toBeUndefined()
 expect(saved).toHaveProperty('storySlug','published-story')
 await expect(restarted.request('/admin/articles/published',input('PATCH',{expectedRevision:3,slug:'renamed-story'}))).rejects.toMatchObject({status:409})
})
test('first story publication keeps the live article permalink and exposes its lock only to editors',async()=>{
 const {demo}=setup()
 const path='/admin/articles/published'
 const updates=[storyUpdate]
 await expect(demo.request(path,input('PATCH',{expectedRevision:1,slug:'draft-slug',storyUpdates:updates}))).rejects.toMatchObject({status:409})
 const ordinaryDraft=await demo.request<Article>(path,input('PATCH',{expectedRevision:1,slug:'draft-slug'}))
 expect(ordinaryDraft.slug).toBe('draft-slug')
 expect(ordinaryDraft).not.toHaveProperty('storySlug')
 await expect(demo.request(path,input('PATCH',{expectedRevision:2,storyUpdates:updates}))).rejects.toMatchObject({status:409})
 const storyDraft=await demo.request<Article>(path,input('PATCH',{expectedRevision:2,slug:'published-story',storyUpdates:updates}))
 expect(storyDraft).not.toHaveProperty('storySlug')
 await demo.request(`${path}/actions`,input('POST',{action:'approve',expectedRevision:3}))
 const published=await demo.request(`${path}/actions`,input('POST',{action:'publish',expectedRevision:3}))
 expect(published).toHaveProperty('storySlug','published-story')
 expect(await demo.request(path)).toHaveProperty('storySlug','published-story')
 expect(await demo.request('/public/articles/published-story')).not.toHaveProperty('storySlug')
 expect(demo.getSnapshot().publications.published).not.toHaveProperty('storySlug')
 await expect(demo.request('/public/articles/draft-slug')).rejects.toMatchObject({status:404})
})
test('approval and publication reject a persisted story draft with a changed permalink',async()=>{
 const {demo,storage,advance}=setup()
 const path='/admin/articles/published'
 await demo.request(path,input('PATCH',{expectedRevision:1,slug:'draft-slug'}))
 const stored=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
 const draft=stored.articles.find((item:Article)=>item.id==='published')
 draft.storyUpdates=[storyUpdate]
 const save=()=>storage.set(DEMO_PIPELINE_KEY,JSON.stringify(stored))
 save()
 await expect(demo.request(`${path}/actions`,input('POST',{action:'approve',expectedRevision:2}))).rejects.toMatchObject({status:409,code:'story_slug_locked'})
 draft.status='approved'
 draft.approvedRevision=2
 save()
 for(const action of ['publish','schedule']){
  await expect(demo.request(`${path}/actions`,input('POST',{action,expectedRevision:2,scheduledAt:'2026-11-15T12:01:00Z'}))).rejects.toMatchObject({status:409,code:'story_slug_locked'})
 }
 draft.scheduledAt='2026-11-15T12:01:00Z'
 save()
 advance(61000)
 expect(demo.getArticle('published-story')?.storyUpdates).toBeUndefined()
 expect(demo.getArticle('draft-slug')).toBeUndefined()
 const reviewed=await demo.request<Article>(path)
 expect(reviewed.status).toBe('review')
 expect(reviewed.scheduledAt).toBeUndefined()
})
test('scheduled timeline publication sorts fractional timestamps and excludes private notes',async()=>{
 const {demo,advance}=setup()
 const older={...storyUpdate,id:'earlier',occurredAt:'2026-11-15T11:00:00.123456Z'}
 const newer={...storyUpdate,id:'later',occurredAt:'2026-11-15T16:00:00.123789+05:00'}
 const tie={...newer,id:'same-time',occurredAt:'2026-11-15T11:00:00.123789Z'}
 const updates=[older,newer,tie]
 const path='/admin/articles/published'
 const draft=await demo.request<Article>(path,input('PATCH',{expectedRevision:1,storyUpdates:updates,reviewNotes:['Private review note']}))
 expect(draft.storyUpdates?.map(update=>update.id)).toEqual(['earlier','later','same-time'])
 await demo.request(`${path}/actions`,input('POST',{action:'approve',expectedRevision:draft.revision}))
 await demo.request(`${path}/actions`,input('POST',{action:'schedule',expectedRevision:draft.revision,scheduledAt:'2026-11-15T12:01:00Z'}))
 expect(demo.getArticle('published-story')?.storyUpdates).toBeUndefined()
 advance(61000)
 const article=demo.getArticle('published-story')
 expect(article?.storyUpdates?.map(update=>update.id)).toEqual(['later','same-time','earlier'])
 expect(article?.reviewNotes).toEqual([])
 expect(article).not.toHaveProperty('storySlug')
 expect(await demo.request(path)).toHaveProperty('storySlug','published-story')
 const withdrawn=await demo.request(`${path}/actions`,input('POST',{action:'unpublish',expectedRevision:draft.revision}))
 expect(withdrawn).toHaveProperty('storySlug','published-story')
 await expect(demo.request(path,input('PATCH',{expectedRevision:draft.revision,slug:'other-story'}))).rejects.toMatchObject({status:409})
})
test('draft creation preserves story fields and requires complete sourced text before approval',async()=>{
 const {demo}=setup()
 const draft=await demo.request<Article>('/admin/articles',input('POST',{
  title:text('Draft with updates'),actionGuide:{...guide,sourceUrl:''},storyUpdates:[storyUpdate],
 }))
 expect(draft.actionGuide?.sourceUrl).toBe('')
 expect(draft.storyUpdates).toEqual([storyUpdate])
 expect(demo.getArticle(draft.id)).toBeUndefined()
 await expect(demo.request(`/admin/articles/${draft.id}/actions`,input('POST',{action:'approve',expectedRevision:1}))).rejects.toMatchObject({status:400})
 await expect(demo.request(`/admin/articles/${draft.id}`,input('PATCH',{expectedRevision:1,storyUpdates:[{...storyUpdate,occurredAt:'2026-11-20'}]}))).rejects.toMatchObject({status:400})
})
test('followed stories are ordered by latest published update and are removable after withdrawal',async()=>{
 const storySeed={...seed,articles:[
  fixture,
  {...fixture,id:'first',slug:'first-story',storyUpdates:[{...storyUpdate,occurredAt:'2026-11-20T19:00:00+05:00'}]},
  {...fixture,id:'second',slug:'second-story',storyUpdates:[storyUpdate]},
  {...fixture,id:'unpublished',slug:'unpublished-story',status:'review' as const,storyUpdates:[storyUpdate]},
 ]}
 const storage=new Map<string,string>()
 const demo=createPipelineDemo(storySeed,storage)
 await demo.request('/account/saved/article/published',input('POST'))
 expect(await demo.request('/account/followed-stories')).toEqual({items:[],unreadUpdateIds:{}})
 await expect(demo.request('/account/followed-stories/published',input('POST'))).rejects.toMatchObject({status:404})
 await expect(demo.request('/account/followed-stories/unpublished',input('POST'))).rejects.toMatchObject({status:404})
 expect(await demo.request('/account/followed-stories/first',input('POST'))).toEqual({ok:true})
 await demo.request('/account/followed-stories/first',input('POST'))
 await demo.request('/account/followed-stories/second',input('POST'))
 await demo.request('/admin/articles/first',input('PATCH',{expectedRevision:1,storyUpdates:[{...storyUpdate,occurredAt:'2026-11-21T00:00:00Z'}]}))
 const restarted=createPipelineDemo(storySeed,storage)
 expect((await restarted.request<{items:Article[]}>('/account/followed-stories')).items.map(item=>item.id)).toEqual(['second','first'])
 const persisted=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
 delete persisted.seenStoryUpdateIds
 storage.set(DEMO_PIPELINE_KEY,JSON.stringify(persisted))
 expect(await restarted.request('/account/followed-stories')).toMatchObject({
  unreadUpdateIds:{first:['call-opens'],second:['call-opens']},
 })
 const originalUser=persisted.session.user
 persisted.session.user=persisted.users.find((user:{id:string})=>user.id==='demo-reader')
 storage.set(DEMO_PIPELINE_KEY,JSON.stringify(persisted))
 expect(await restarted.request('/account/followed-stories')).toEqual({items:[],unreadUpdateIds:{}})
 persisted.session.user=originalUser
 storage.set(DEMO_PIPELINE_KEY,JSON.stringify(persisted))
 await restarted.request('/admin/articles/second/actions',input('POST',{action:'unpublish',expectedRevision:1}))
 expect((await restarted.request<{items:Article[]}>('/account/followed-stories')).items.map(item=>item.id)).toEqual(['first'])
 expect(await restarted.request('/account/followed-stories/second',input('DELETE'))).toEqual({ok:true})
 await restarted.request('/admin/articles/second/actions',input('POST',{action:'approve',expectedRevision:1}))
 await restarted.request('/admin/articles/second/actions',input('POST',{action:'publish',expectedRevision:1}))
 expect((await restarted.request<{items:Article[]}>('/account/followed-stories')).items.map(item=>item.id)).toEqual(['first'])
 await restarted.request('/auth/logout',input('POST'))
 await expect(restarted.request('/account/followed-stories')).rejects.toMatchObject({status:401})
})
test('read receipts track only submitted published update IDs across revising, following again and restarting',async()=>{
 const storySeed={...seed,articles:[{...fixture,storyUpdates:[storyUpdate]}]}
 const storage=new Map<string,string>()
 const demo=createPipelineDemo(storySeed,storage)
 const follows='/account/followed-stories',follow=`${follows}/published`,read=`${follow}/read`
 const unread=async(engine=demo)=>(await engine.request<FollowedStories>(follows)).unreadUpdateIds
 await expect(demo.request(read,input('POST',{updateIds:[storyUpdate.id]}))).rejects.toMatchObject({status:404})
 await demo.request(follow,input('POST'))
 expect(await unread()).toEqual({published:[]})

 const actions='/admin/articles/published/actions'
 const newer={...storyUpdate,id:'deadline-extended',occurredAt:'2026-11-25T12:00:00Z'}
 const latest={...storyUpdate,id:'deadline-closed',occurredAt:'2026-11-26T12:00:00Z'}
 const draft=await demo.request<Article>('/admin/articles/published',input('PATCH',{
  expectedRevision:1,storyUpdates:[{...storyUpdate,body:text('Corrected source text')},newer,latest],
 }))
 expect(await demo.request(read,input('POST',{updateIds:[latest.id]}))).toEqual({ok:true})
 await demo.request(actions,input('POST',{action:'approve',expectedRevision:draft.revision}))
 await demo.request(actions,input('POST',{action:'publish',expectedRevision:draft.revision}))
 await demo.request(follow,input('POST'))
 expect(await unread()).toEqual({published:['deadline-closed','deadline-extended']})
 await expect(demo.request(read,input('POST',{updateIds:['invalid ID']}))).rejects.toMatchObject({status:400})
 await expect(demo.request(read,input('POST',{updateIds:[latest.id],all:true}))).rejects.toMatchObject({status:400})
 expect(await demo.request(read,input('POST',{updateIds:[newer.id,newer.id,'unknown-update']}))).toEqual({ok:true})
 const restarted=createPipelineDemo(storySeed,storage)
 expect(await unread(restarted)).toEqual({published:['deadline-closed']})
 expect(await restarted.request(read,input('POST',{updateIds:[latest.id]}))).toEqual({ok:true})
 expect(await unread(restarted)).toEqual({published:[]})
 await restarted.request(follow,input('DELETE'))
 expect(await unread(restarted)).toEqual({})
})
test('reminders survive revisions and withdrawal until an editor completes them',async()=>{
 const {demo,storage}=setup()
 const article='/admin/articles/published',reminder=`${article}/follow-up`,queue='/admin/story-follow-ups'
 const reminderData={dueDate:'2026-11-19',assigneeId:'demo-author'}
 const draft=await demo.request<Article>(article,input('PATCH',{expectedRevision:1,storyUpdates:[storyUpdate]}))
 expect(await demo.request(reminder)).toEqual({item:null,eligible:true})
 expect(await demo.request(reminder,input('PUT',reminderData))).toEqual({item:{
  articleId:'published',title:fixture.title,assigneeId:'demo-author',assigneeName:'Редактор демоверсии',
  dueDate:'2026-11-19',updatedAt:'2026-11-15T12:00:00.000Z',
 }})
 expect(await demo.request(article)).toEqual(draft)
 expect(demo.getArticle('published-story')).not.toHaveProperty('followUp')
 await demo.request(`${article}/actions`,input('POST',{action:'approve',expectedRevision:draft.revision}))
 await demo.request(`${article}/actions`,input('POST',{action:'publish',expectedRevision:draft.revision}))
 const revised=await demo.request<Article>(article,input('PATCH',{expectedRevision:draft.revision,storyUpdates:[],title:text('New draft title')}))
 expect(await demo.request(reminder,input('PUT',{...reminderData,dueDate:'2026-11-21'}))).toMatchObject({item:{dueDate:'2026-11-21',title:revised.title}})
 await demo.request(`${article}/actions`,input('POST',{action:'unpublish',expectedRevision:revised.revision}))
 expect(await demo.request(reminder)).toMatchObject({item:{dueDate:'2026-11-21'},eligible:false})

 const earlier=await demo.request<Article>('/admin/articles',input('POST',{title:text('Draft story'),storyUpdates:[storyUpdate]}))
 await demo.request(`/admin/articles/${earlier.id}/follow-up`,input('PUT',{...reminderData,dueDate:'2026-11-17'}))
 const restarted=createPipelineDemo(seed,storage)
 const list=await restarted.request<{items:Array<{articleId:string}>;total:number}>(queue)
 expect(list.total).toBe(2)
 expect(list.items.map(item=>item.articleId)).toEqual([earlier.id,'published'])
 expect(await restarted.request(reminder,input('DELETE'))).toEqual({ok:true})
 expect(await restarted.request(reminder,input('DELETE'))).toEqual({ok:true})
 expect(await restarted.request(reminder)).toEqual({item:null,eligible:false})
 const audit=await restarted.request<{items:Array<{action:string;targetId:string}>}>('/admin/audit')
 expect(audit.items.filter(item=>item.targetId==='published'&&item.action.startsWith('story_follow_up.')).map(item=>item.action)).toEqual([
  'story_follow_up.complete','story_follow_up.set','story_follow_up.set',
 ])
})
test('reminders require publication permissions and a valid date on a story',async()=>{
 const {demo}=setup()
 const queue='/admin/story-follow-ups',published='/admin/articles/published/follow-up'
 const reminderData={dueDate:'2026-11-19',assigneeId:'demo-author'}
 await expect(demo.request(published,input('PUT',reminderData))).rejects.toMatchObject({status:400})
 const world=await demo.request<Article>('/admin/articles',input('POST',{title:text('World story'),storyUpdates:[storyUpdate]}))
 const economy=await demo.request<Article>('/admin/articles',input('POST',{title:text('Eco story'),layer:'eco',storyUpdates:[storyUpdate]}))
 const worldPath=`/admin/articles/${world.id}/follow-up`,economyPath=`/admin/articles/${economy.id}/follow-up`
 await expect(demo.request(worldPath,input('PUT',{...reminderData,expectedRevision:1}))).rejects.toMatchObject({status:400})
 for(const dueDate of ['2026-02-29','2026-11-31','0000-11-23','2026-11-19T09:00:00Z']){
  await expect(demo.request(worldPath,input('PUT',{...reminderData,dueDate}))).rejects.toMatchObject({status:400})
 }
 for(const assigneeId of ['demo-reader','demo-editor','missing-user']){
  await expect(demo.request(worldPath,input('PUT',{...reminderData,assigneeId}))).rejects.toMatchObject({status:400})
 }
 await demo.request('/admin/users/demo-editor',input('PATCH',{verticals:['ai']}))
 await demo.request(economyPath,input('PUT',reminderData))
 await demo.request('/auth/dev-login',input('POST',{role:'editor'}))
 expect(await demo.request(worldPath,input('PUT',{...reminderData,assigneeId:'demo-editor',dueDate:'2028-02-29'}))).toMatchObject({item:{assigneeId:'demo-editor'}})
 expect(await demo.request(queue)).toMatchObject({total:1,items:[{articleId:world.id}]})
 await expect(demo.request(economyPath)).rejects.toMatchObject({status:403})
 await expect(demo.request(economyPath,input('DELETE'))).rejects.toMatchObject({status:403})
 await demo.request('/auth/dev-login',input('POST',{role:'author'}))
 await expect(demo.request(queue)).rejects.toMatchObject({status:403})
 await expect(demo.request(worldPath,input('PUT',reminderData))).rejects.toMatchObject({status:403})
})
test('schedule audit entries are restricted to administrators and chief editors',async()=>{
 const {demo}=setup()
 const article=await demo.request<Article>('/admin/articles',input('POST',{title:text('Story'),storyUpdates:[storyUpdate]}))
 const dueDate='2026-11-19',assigneeId='demo-author'
 await demo.request(`/admin/articles/${article.id}/follow-up`,input('PUT',{dueDate,assigneeId}))
 const audit='/admin/audit'
 for(const role of ['author','editor']){
  await demo.request('/auth/dev-login',input('POST',{role}))
  await expect(demo.request(audit)).rejects.toMatchObject({status:403})
 }
 const auditEntry={targetId:article.id,action:'story_follow_up.set',detail:{dueDate,assigneeId}}
 await demo.request('/auth/dev-login',input('POST',{role:'admin'}))
 expect(await demo.request(audit)).toMatchObject({items:expect.arrayContaining([expect.objectContaining(auditEntry)])})
 await demo.request('/admin/users/demo-author',input('PATCH',{roles:['chief_editor']}))
 expect(await demo.request(audit)).toMatchObject({items:expect.arrayContaining([expect.objectContaining(auditEntry)])})
})
test('published snapshots stay immutable until the edited concrete revision is approved and published',async()=>{
 const {demo}=setup();let a=await demo.request<Article>('/admin/articles/published',{...input('PATCH',{expectedRevision:1,title:text('Изменено')})});expect(a.revision).toBe(2);expect(a.approvedRevision).toBeUndefined();expect(demo.getPublishedArticles().find(a=>a.id==='published')?.title.ru).toBe(fixture.title.ru)
 await expect(demo.request('/admin/articles/published/actions',input('POST',{action:'publish',expectedRevision:2}))).rejects.toMatchObject({status:409})
 a=await demo.request('/admin/articles/published/actions',input('POST',{action:'approve',expectedRevision:2}));expect(a.approvedRevision).toBe(2)
 await expect(demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Stale')}))).rejects.toMatchObject({status:409})
 a=await demo.request('/admin/articles/published/actions',input('POST',{action:'publish',expectedRevision:a.revision}));expect(demo.getArticle('published-story')?.title.ru).toBe('Изменено');expect(a.publishedRevision).toBe(a.revision)
 const exposed=demo.getPublishedArticles();exposed[0].title.ru='Caller mutation';expect(demo.getArticle('published')?.title.ru).toBe('Изменено')
 const withdrawn=await demo.request<Article>('/admin/articles/published/actions',input('POST',{action:'unpublish',expectedRevision:a.revision}));expect(withdrawn.status).toBe('unpublished');expect(demo.getArticle('published')).toBeUndefined()
})
test('source collection → evidence → trilingual draft → review → approval → publication → simulated digest delivery',async()=>{
 const {demo,advance}=setup();const source=await demo.request<Source>('/admin/sources',input('POST',{name:'Research feed',url:'https://example.org/feed',type:'rss',locale:'en',topics:['ai'],intervalMinutes:30,active:true}));const run=await demo.request<Job>(`/admin/sources/${source.id}/run`,input('POST'));expect(run.status).toBe('queued');advance();const bank=await demo.request<{items:Story[]}>('/admin/bank');const story=bank.items.find(s=>s.sourceId===source.id)!;expect(story.sources.length).toBeGreaterThan(0);expect(story.score).toBeGreaterThan(0)
 let draft=await demo.request<Article>(`/admin/bank/${story.id}/draft`,input('POST'));advance();draft=await demo.request(`/admin/articles/${draft.id}`);expect(draft.status).toBe('review');expect(draft.researchBrief?.retrievedEvidence.length).toBeGreaterThan(0);for(const l of ['ru','kk','en'] as const)expect(draft.body[l].length).toBeGreaterThan(20);expect(draft.opportunityIds).toContain('op');expect(demo.getArticle(draft.id)).toBeUndefined()
 const research=await demo.request<{items:Job[]}>(`/admin/articles/${draft.id}/research`);expect(research.items[0].checkpoints?.review.status).toBe('succeeded')
 draft=await demo.request(`/admin/articles/${draft.id}/actions`,input('POST',{action:'approve',expectedRevision:draft.revision}));draft=await demo.request(`/admin/articles/${draft.id}/actions`,input('POST',{action:'publish',expectedRevision:draft.revision}));expect(demo.getArticle(draft.id)).toBeDefined()
 let digest=await demo.request<Digest>('/admin/digests/preview',input('POST',{locale:'en',kind:'channel'}));expect(digest.articles.some(a=>a.id===draft.id)).toBe(true);await expect(demo.request(`/admin/digests/${digest.id}/send`,input('POST'))).rejects.toMatchObject({status:409});digest=await demo.request(`/admin/digests/${digest.id}/approve`,input('POST'));await demo.request(`/admin/digests/${digest.id}/send`,input('POST'));const deliveries=await demo.request<{items:Array<{chatId:string;simulated:boolean}>}>('/admin/deliveries');expect(deliveries.items.every(d=>d.simulated&&d.chatId.startsWith('demo:'))).toBe(true)
})
test('manual URL collection and retries are idempotent across repeated clicks and restarts',async()=>{
 const {demo,storage,advance}=setup();const first=await demo.request<Job>('/admin/bank/collect',input('POST',{url:'https://example.org/manual',locale:'ru',topics:['ai']}));const repeat=await demo.request<Job>('/admin/bank/collect',input('POST',{url:'https://example.org/manual',locale:'ru',topics:['ai']}));expect(repeat.id).toBe(first.id);await demo.request(`/admin/jobs/${first.id}/cancel`,input('POST'));const retry=await demo.request<Job>(`/admin/jobs/${first.id}/retry`,input('POST'));const again=await demo.request<Job>(`/admin/jobs/${first.id}/retry`,input('POST'));expect(again.attempts).toBe(retry.attempts);advance();const bank=await demo.request<{items:Story[]}>('/admin/bank');expect(bank.items.filter(s=>s.url==='https://example.org/manual')).toHaveLength(1);const other=createPipelineDemo(seed,storage,()=>Date.parse('2026-11-15T13:00:00Z'));expect((await other.request<{items:Story[]}>('/admin/bank')).items.filter(s=>s.url==='https://example.org/manual')).toHaveLength(1)
})
test('restricted roles cannot approve, publish or edit someone else; default session aligns with an author',async()=>{
 const {demo}=setup();const me=await demo.request<{user:{id:string;roles:string[]}}>('/auth/me');expect(me.user.roles).toContain('chief_editor');expect((await demo.request<{items:Array<{id:string}>}>('/admin/authors')).items.some(a=>a.id===me.user.id)).toBe(true)
 await demo.request('/auth/dev-login',input('POST',{role:'reader'}));await expect(demo.request('/admin/articles')).rejects.toMatchObject({status:403});await demo.request('/auth/dev-login',input('POST',{role:'author'}));await expect(demo.request('/admin/articles/published/actions',input('POST',{action:'approve',expectedRevision:1}))).rejects.toMatchObject({status:403});await demo.request('/auth/logout',input('POST'));await expect(demo.request('/admin/sources')).rejects.toMatchObject({status:401});expect(demo.getPublishedArticles().length).toBeGreaterThan(0)
})
test('editing an approved article invalidates approval and rework runs without mutating public text',async()=>{
 const {demo,advance}=setup();let a=await demo.request<Article>('/admin/articles/published/actions',input('POST',{action:'rework',expectedRevision:1,note:'Объяснить влияние на читателя'}));expect(a.status).toBe('rework');advance();a=await demo.request(`/admin/articles/${a.id}`);expect(a.status).toBe('review');expect(demo.getArticle(a.id)?.body.ru).toBe(fixture.body.ru);a=await demo.request(`/admin/articles/${a.id}/actions`,input('POST',{action:'approve',expectedRevision:a.revision}));a=await demo.request(`/admin/articles/${a.id}`,input('PATCH',{expectedRevision:a.revision,lead:text('Новый лид')}));expect(a.approvedRevision).toBeUndefined();await expect(demo.request(`/admin/articles/${a.id}/actions`,input('POST',{action:'publish',expectedRevision:a.revision}))).rejects.toMatchObject({status:409})
})
test('independent storage isolates visitors; existing edits survive restart and partial seeds',async()=>{
 const {demo,storage}=setup();await demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Локально')}));expect(storage.has(DEMO_PIPELINE_KEY)).toBe(true);const restarted=createPipelineDemo({articles:[],categories:[],opportunities:[]},storage);expect((await restarted.request<Article>('/admin/articles/published')).title.ru).toBe('Локально');expect(setup().demo.getArticle('published')?.title.ru).toBe(fixture.title.ru);demo.reset();expect((await demo.request<Article>('/admin/articles/published')).title.ru).toBe(fixture.title.ru)
})
test('plain reads neither rewrite storage nor emit synthetic change events',async()=>{
 let writes=0;const data=new Map<string,string>();const storage={getItem:(k:string)=>data.get(k)??null,setItem:(k:string,v:string)=>{writes++;data.set(k,v)},removeItem:(k:string)=>{data.delete(k)}};const demo=createPipelineDemo(seed,storage);await demo.request('/admin/articles');demo.getSnapshot();demo.getPublishedArticles();expect(writes).toBe(0);await demo.request('/admin/articles',input('POST',{title:text('Draft'),categoryId:'ai'}));expect(writes).toBe(1);await demo.request('/admin/jobs');demo.getSnapshot();expect(writes).toBe(1)
})
test('settings discard entered provider keys and unknown/external endpoints fail explicitly',async()=>{
 const {demo,storage}=setup();await demo.request('/admin/settings',input('PATCH',{ai:{apiKey:'sk-real-secret',endpoint:'https://provider.example',model:'real-model'}}));expect(storage.get(DEMO_PIPELINE_KEY)).not.toContain('sk-real-secret');const settings=await demo.request<{ai:{model:string;keyConfigured:boolean}}>('/admin/settings');expect(settings.ai.model).toContain('demo');await expect(demo.request('/admin/missing',input('POST'))).rejects.toBeInstanceOf(ApiError);await expect(demo.request('/auth/google/start')).rejects.toMatchObject({status:501});await expect(demo.request('/admin/sources',input('POST',{name:'bad',url:'javascript:alert(1)',type:'rss',intervalMinutes:30}))).rejects.toMatchObject({status:400})
})
test('auxiliary CRUD supports authors, opportunities, launches, media, pages and newspapers',async()=>{
 const {demo}=setup();const author=await demo.request<{id:string}>('/admin/authors',input('POST',{name:'Журналист',bio:text('Биография')}));expect(author.id).toBeTruthy();const opportunity=await demo.request<{id:string}>('/admin/opportunities',input('POST',{...seed.opportunities[0],id:undefined,title:text('Возможность')}));expect(opportunity.id).not.toBe('op')
 const launch=await demo.request<{id:string;revision:number}>('/admin/launches',input('POST',{slug:'demo-launch',name:'Test',tagline:text('Tag'),description:text('Desc'),team:text('Team'),url:'https://example.org',launchDate:'2026-11-15',country:'KZ',categoryId:'ai'}));await demo.request(`/admin/launches/${launch.id}`,input('PATCH',{expectedRevision:launch.revision,status:'published'}));expect((await demo.request<{items:unknown[]}>('/public/launches')).items.length).toBeGreaterThan(0)
 const form=new FormData();form.set('file',new File(['image bytes'],'photo.png',{type:'image/png'}));form.set('credit','Author');form.set('rights','Demo use');const media=await demo.request<{id:string;url:string;revision:number}>('/admin/media',{method:'POST',body:form});expect(media.url.startsWith('data:')).toBe(true);await demo.request(`/admin/media/${media.id}`,input('PATCH',{expectedRevision:media.revision,caption:'Updated'}));const issue=await demo.request<{id:string;revision:number}>('/admin/newspapers',input('POST',{slug:'02',number:'02',date:'2026-11',title:'Demo issue',locale:'ru',items:[{id:'entry',articleId:'published'}],cover:fixture.media}));await demo.request(`/admin/newspapers/${issue.id}/publish`,input('POST',{expectedRevision:issue.revision}));expect((await demo.request<{entries:unknown[]}>('/public/newspapers/02')).entries).toHaveLength(1)
})
test('unknown action suffixes and unsupported methods do not fall through to generic success',async()=>{
 const {demo}=setup();for(const [method,path]of [['POST','/admin/articles/published/actions/extra'],['POST','/admin/digests/preview/extra'],['GET','/auth/me/extra'],['DELETE','/admin/articles/published'],['GET','/public/articles/published-story/extra']])await expect(demo.request(path,input(method,{action:'submit',expectedRevision:1}))).rejects.toMatchObject({status:404})
})
test('new seeds add missing items and fields without reverting drafts, published snapshots or deletions',async()=>{
 const {demo,storage}=setup();await demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Edited')}));await demo.request('/admin/opportunities/op',input('DELETE'));const extra={...fixture,id:'new-seed',slug:'new-seed'};const next=createPipelineDemo({...seed,articles:[{...fixture,title:text('Changed upstream')},extra]},storage);expect((await next.request<Article>('/admin/articles/published')).title.ru).toBe('Edited');expect(next.getArticle('published')?.title.ru).toBe(fixture.title.ru);expect(next.getArticle('new-seed')).toBeDefined();expect((await next.request<{items:unknown[]}>('/admin/opportunities')).items).toHaveLength(0)
 const raw=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!);delete raw.articles[0].tags;storage.set(DEMO_PIPELINE_KEY,JSON.stringify(raw));expect((await next.request<Article>(`/admin/articles/${raw.articles[0].id}`)).tags).toBeDefined()
})
test('scheduled article may be rescheduled, published, withdrawn and approved again',async()=>{
 const {demo,storage,advance}=setup()
 const path='/admin/articles/demo-editorial-approved'
 const act=(action:string,scheduledAt?:string)=>demo.request<Article>(path+'/actions',input('POST',{action,expectedRevision:1,scheduledAt}))
 const scheduled=await act('schedule','2026-11-15T12:01:00Z')
 expect(scheduled.status).toBe('scheduled')
 expect(scheduled.approvedRevision).toBe(1)
 expect(scheduled.scheduledAt).toBe('2026-11-15T12:01:00.000Z')
 expect(demo.getArticle(scheduled.id)).toBeUndefined()
 const analytics=await demo.request<{statuses:Record<string,number>}>('/admin/analytics')
 expect(analytics.statuses.scheduled).toBe(1)
 expect(analytics.statuses.unpublished).toBe(0)
 expect((await act('schedule','2026-11-15T12:02:00Z')).status).toBe('scheduled')
 advance(61000)
 expect(demo.getArticle(scheduled.id)).toBeUndefined()
 expect((await act('publish')).status).toBe('published')
 expect(demo.getArticle(scheduled.id)?.status).toBe('published')
 const withdrawn=await act('unpublish')
 expect(withdrawn.status).toBe('unpublished')
 expect(withdrawn.approvedRevision).toBeUndefined()
 expect(withdrawn.publishedRevision).toBeUndefined()
 expect(withdrawn.scheduledAt).toBeUndefined()
 expect(demo.getArticle(scheduled.id)).toBeUndefined()
 await expect(act('publish')).rejects.toMatchObject({status:409})
 const reopened=createPipelineDemo(seed,storage,()=>Date.parse('2026-11-15T12:01:01Z'))
 expect((await reopened.request<Article>(path)).status).toBe('unpublished')
 expect((await act('approve')).status).toBe('approved')
 expect((await act('publish')).status).toBe('published')
})
test('submitting or requesting changes cancels publication time; redundant approval cannot cancel it',async()=>{
 const {demo,advance}=setup()
 const path='/admin/articles/demo-editorial-approved'
 const act=(action:string)=>demo.request<Article>(path+'/actions',input('POST',{action,expectedRevision:1,scheduledAt:'2026-11-15T12:01:00Z',note:'Editorial change'}))
 await expect(act('unpublish')).rejects.toMatchObject({status:409})
 await act('schedule')
 await expect(act('approve')).rejects.toMatchObject({status:409})
 expect((await demo.request<Article>(path)).status).toBe('scheduled')
 const review=await act('submit')
 expect(review.scheduledAt).toBeUndefined()
 expect(review.approvedRevision).toBeUndefined()
 await act('approve')
 await act('schedule')
 const reworked=await act('rework')
 expect(reworked.scheduledAt).toBeUndefined()
 expect(reworked.approvedRevision).toBeUndefined()
 advance(61000)
 expect(demo.getArticle(reworked.id)).toBeUndefined()
})
test('legacy scheduled and explicitly withdrawn demo records are recovered without changing unrelated review drafts',async()=>{
 const {demo,storage}=setup()
 const path='/admin/articles/demo-editorial-approved'
 await demo.request(path+'/actions',input('POST',{action:'schedule',expectedRevision:1,scheduledAt:'2026-11-15T12:01:00Z'}))
 await demo.request('/admin/articles/published/actions',input('POST',{action:'unpublish',expectedRevision:1}))
 const raw=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
 raw.articles.find((a:Article)=>a.id==='demo-editorial-approved').status='approved'
 raw.articles.find((a:Article)=>a.id==='published').status='review'
 storage.set(DEMO_PIPELINE_KEY,JSON.stringify(raw))
 const reopened=createPipelineDemo(seed,storage,()=>Date.parse('2026-11-15T12:00:00Z'))
 expect((await reopened.request<Article>(path)).status).toBe('scheduled')
 expect((await reopened.request<Article>('/admin/articles/published')).status).toBe('unpublished')
 expect((await reopened.request<Article>('/admin/articles/demo-editorial-review')).status).toBe('review')
 expect(reopened.getArticle('published')).toBeUndefined()
})
test('legacy schedule on a live revision retains its public snapshot',async()=>{
 const {demo,storage}=setup()
 await demo.request('/admin/settings',input('PATCH',{digestHour:10}))
 const raw=JSON.parse(storage.get(DEMO_PIPELINE_KEY)!)
 const live=raw.articles.find((a:Article)=>a.id==='published')
 live.scheduledAt='2026-11-15T12:01:00Z'
 storage.set(DEMO_PIPELINE_KEY,JSON.stringify(raw))
 const reopened=createPipelineDemo(seed,storage,()=>Date.parse('2026-11-15T12:00:00Z'))
 expect((await reopened.request<Article>('/admin/articles/published')).status).toBe('scheduled')
 expect(reopened.getArticle('published')?.status).toBe('published')
})
test('scheduled publication completes from the injected clock without timers or storage writes on reads',async()=>{
 const {demo,storage,advance}=setup();let a=await demo.request<Article>('/admin/articles/demo-editorial-approved/actions',input('POST',{action:'schedule',expectedRevision:1,scheduledAt:'2026-11-15T12:01:00Z'}));expect(demo.getArticle(a.id)).toBeUndefined();const before=storage.get(DEMO_PIPELINE_KEY);advance(61000);expect(demo.getArticle(a.id)?.status).toBe('published');expect(storage.get(DEMO_PIPELINE_KEY)).toBe(before);expect(createPipelineDemo(seed,storage,()=>Date.parse('2026-11-15T12:02:00Z')).getArticle(a.id)?.status).toBe('published')
})
test('page edits and unpublishes respect frozen snapshots across restart',async()=>{
 const storage=new Map<string,string>();const content={ru:{title:'About',body:'Old',second:'',values:[]},kk:{title:'About',body:'Old',second:'',values:[]},en:{title:'About',body:'Old',second:'',values:[]}};const enriched={...seed,pages:[{id:'page-about',slug:'about' as const,content,status:'published' as const,revision:1,publishedRevision:1}]};const demo=createPipelineDemo(enriched,storage);const updated=structuredClone(content);updated.ru.body='Changed';let page=await demo.request<{revision:number}>('/admin/pages/about',input('PATCH',{expectedRevision:1,content:updated}));expect((await demo.request<{content:typeof content}>('/public/pages/about')).content.ru.body).toBe('Old');await demo.request('/admin/pages/about/publish',input('POST',{expectedRevision:page.revision}));expect((await demo.request<{content:typeof content}>('/public/pages/about')).content.ru.body).toBe('Changed');await demo.request('/admin/pages/about/unpublish',input('POST',{expectedRevision:page.revision}));await expect(createPipelineDemo(enriched,storage).request('/public/pages/about')).rejects.toMatchObject({status:404})
})
test('media safeguards reject oversized files and preserve referenced assets',async()=>{
 const {demo}=setup();const form=new FormData();form.set('file',new File([new Uint8Array(2*1024*1024+1)],'large.png',{type:'image/png'}));form.set('credit','Author');form.set('rights','Demo use');await expect(demo.request('/admin/media',{method:'POST',body:form})).rejects.toMatchObject({status:413});const media=await demo.request<{items:Array<{id:string;revision:number}>}>('/admin/media');await expect(demo.request(`/admin/media/${media.items[0].id}`,input('DELETE',{expectedRevision:media.items[0].revision}))).rejects.toMatchObject({status:409})
})
test('job projections are deterministic and read-only, and rework never overwrites a later manual edit',async()=>{
 const {demo,advance,storage}=setup();await demo.request('/admin/articles/published/actions',input('POST',{action:'rework',expectedRevision:1,note:'Clarify'}));await demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,body:text('Manual edit wins')}));const stored=storage.get(DEMO_PIPELINE_KEY);advance();expect((await demo.request<Article>('/admin/articles/published')).body.ru).toBe('Manual edit wins');const jobs=await demo.request<{items:Job[]}>('/admin/jobs');expect(jobs.items.find(j=>j.kind==='rework')?.status).toBe('failed');expect(storage.get(DEMO_PIPELINE_KEY)).toBe(stored)
})
test('restricted-storage fallback stays usable within the browser session',async()=>{
 const blocked={getItem:()=>{throw new Error('Blocked')},setItem:()=>{throw new Error('Blocked')}};const demo=createPipelineDemo(seed,blocked);await demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Fallback edit')}));expect((await demo.request<Article>('/admin/articles/published')).title.ru).toBe('Fallback edit')
})
test('repeated publication HTML is stored once, round-trips intact, and demo settings use a valid URL input',async()=>{
 const storage=new Map<string,string>();const largeHtml='<p>Demo original newspaper content.</p>'.repeat(5000);const content={ru:{title:'About',body:largeHtml,second:'',values:[]},kk:{title:'About',body:'',second:'',values:[]},en:{title:'About',body:'',second:'',values:[]}};const enriched={...seed,pages:[{id:'page-about',slug:'about' as const,content,status:'published' as const,revision:1,publishedRevision:1}]};const demo=createPipelineDemo(enriched,storage);const settings=await demo.request<{ai:{endpoint:string}}>('/admin/settings');expect(new URL(settings.ai.endpoint).protocol).toBe('https:');await demo.request('/admin/settings',input('PATCH',{digestHour:10}));expect(storage.get(DEMO_PIPELINE_KEY)!.length).toBeLessThan(largeHtml.length*2);expect((await createPipelineDemo(enriched,storage).request<{content:typeof content}>('/public/pages/about')).content.ru.body).toBe(largeHtml)
})
test('stale inverse merges cannot hide both stories, and merge/split advance evidence versions with a fixed clock',async()=>{
 const {demo}=setup();const bank=await demo.request<{items:Story[]}>('/admin/bank');const [left,right]=bank.items;const evidence=(id:string)=>demo.request<{version:string;story:Story}>(`/admin/bank/${id}/evidence`);const beforeLeft=await evidence(left.id),beforeRight=await evidence(right.id);const merge=await demo.request<{id:string}>(`/admin/bank/${left.id}/merge`,input('POST',{otherId:right.id,expectedUpdatedAt:beforeLeft.version,otherExpectedUpdatedAt:beforeRight.version}));await expect(demo.request(`/admin/bank/${right.id}/merge`,input('POST',{otherId:left.id,expectedUpdatedAt:beforeRight.version,otherExpectedUpdatedAt:beforeLeft.version}))).rejects.toMatchObject({status:409});expect((await demo.request<{items:Story[]}>('/admin/bank')).items.map(s=>s.id)).toContain(left.id)
 const afterLeft=await evidence(left.id),afterRight=await evidence(right.id);expect(afterLeft.version).not.toBe(beforeLeft.version);expect(afterRight.version).not.toBe(beforeRight.version);await expect(demo.request(`/admin/bank/${left.id}/merge`,input('POST',{otherId:right.id,expectedUpdatedAt:afterLeft.version,otherExpectedUpdatedAt:afterRight.version}))).rejects.toMatchObject({status:409});await expect(demo.request(`/admin/bank/${right.id}/draft`,input('POST'))).rejects.toMatchObject({status:409});await expect(demo.request(`/admin/bank/merges/${merge.id}/split`,input('POST',{expectedLeftVersion:beforeLeft.version,expectedRightVersion:beforeRight.version}))).rejects.toMatchObject({status:409})
 await demo.request(`/admin/bank/merges/${merge.id}/split`,input('POST',{expectedLeftVersion:afterLeft.version,expectedRightVersion:afterRight.version}));expect((await evidence(left.id)).version).not.toBe(afterLeft.version);expect((await evidence(right.id)).version).not.toBe(afterRight.version);expect((await demo.request<{items:Story[]}>('/admin/bank')).items).toHaveLength(bank.items.length);await expect(demo.request(`/admin/bank/merges/${merge.id}/undo`,input('POST'))).rejects.toMatchObject({status:409})
})
test('opportunity rights review confirmation persists and rejects nonboolean values',async()=>{
 const {demo,storage}=setup();await demo.request('/admin/opportunities/op',input('PATCH',{needsRightsReview:true}));const updated=await demo.request<{needsRightsReview:boolean}>('/admin/opportunities/op',input('PATCH',{needsRightsReview:false}));expect(updated.needsRightsReview).toBe(false);expect((await createPipelineDemo(seed,storage).request<{items:Array<{id:string;needsRightsReview:boolean}>}>('/admin/opportunities')).items.find(o=>o.id==='op')?.needsRightsReview).toBe(false);await expect(demo.request('/admin/opportunities/op',input('PATCH',{needsRightsReview:'false'}))).rejects.toMatchObject({status:400})
})
test('an upload finishing later cannot overwrite edits committed while the file was being read',async()=>{
 const {demo,storage}=setup();let release!:()=>void,reading!:()=>void;const gate=new Promise<void>(resolve=>{release=resolve}),started=new Promise<void>(resolve=>{reading=resolve});const form=new FormData();form.set('file',new File(['delayed image'],'delayed.png',{type:'image/png'}));form.set('credit','Author');form.set('rights','Demo use');const original=Blob.prototype.arrayBuffer;Blob.prototype.arrayBuffer=async function(){const bytes=await original.call(this);reading();await gate;return bytes}
 try{const upload=demo.request<{id:string}>('/admin/media',{method:'POST',body:form});await started;const other=createPipelineDemo(seed,storage);await other.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Concurrent edit survives')}));release();const asset=await upload;const article=await demo.request<Article>('/admin/articles/published');expect(article.title.ru).toBe('Concurrent edit survives');expect(article.revision).toBe(2);expect((await demo.request<{items:Array<{id:string}>}>('/admin/media')).items.some(item=>item.id===asset.id)).toBe(true);await expect(demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Stale edit')}))).rejects.toMatchObject({status:409})}finally{release();Blob.prototype.arrayBuffer=original}
})
test('runtime storage quota failure rejects without losing persisted state or stranding future edits in memory',async()=>{
 const records=new Map<string,string>();const storage={getItem:(key:string)=>records.get(key)??null,setItem:(key:string,value:string)=>{if(value.length>5*1024*1024)throw new DOMException('Storage full','QuotaExceededError');records.set(key,value)}};const demo=createPipelineDemo(seed,storage);const form=(fill:number)=>{const f=new FormData();f.set('file',new File([new Uint8Array(2*1024*1024).fill(fill)],`photo-${fill}.png`,{type:'image/png'}));f.set('credit','Author');f.set('rights','Demo use');return f};const first=await demo.request<{id:string}>('/admin/media',{method:'POST',body:form(1)});const committed=records.get(DEMO_PIPELINE_KEY);await expect(demo.request('/admin/media',{method:'POST',body:form(2)})).rejects.toMatchObject({status:507});expect(records.get(DEMO_PIPELINE_KEY)).toBe(committed);const current=await demo.request<{items:Array<{id:string}>}>('/admin/media');expect(current.items.filter(m=>m.id===first.id)).toHaveLength(1);expect(current.items).toHaveLength(2);expect((await createPipelineDemo(seed,storage).request<{items:unknown[]}>('/admin/media')).items).toHaveLength(current.items.length);await demo.request('/admin/articles/published',input('PATCH',{expectedRevision:1,title:text('Saved after quota rejection')}));expect((await createPipelineDemo(seed,storage).request<Article>('/admin/articles/published')).title.ru).toBe('Saved after quota rejection')
})
})
