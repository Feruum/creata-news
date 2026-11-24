import type { Article } from '@creata/contracts'
import { auth, auxiliary, prepareMediaUpload, uploadMedia } from './auxiliary'
import { editorial } from './editorial'
import { advanceJobs, hermes } from './hermes'
import { clone, fail, makeContext, requireRole, type Body, type DemoClock, type DemoSeed, type DemoStorage } from './model'
import { createStore } from './store'
import { supportsRoute } from './routes'
export { DEMO_PIPELINE_KEY, DEMO_PIPELINE_EVENT } from './model'
export { followedStoryRequest, isFollowableStory, listFollowedStories } from './story-content'
export type { DemoSeed, DemoState, DemoStorage, DemoClock, DemoAudit, DemoDelivery, FollowedStoryState } from './model'

/** An isolated local API. No fetch, workers, credentials, or external sends. */
export function createPipelineDemo(seed:DemoSeed,storage?:DemoStorage,now?:DemoClock) {
 const store=createStore(seed,storage,now)
 const context=()=>{const c=makeContext(store.read(),store.clock());advanceJobs(c);return c}
 async function request<T>(path:string,init:RequestInit={}):Promise<T> {
  if(init.signal?.aborted)throw new DOMException('Aborted','AbortError')
  if(!path.startsWith('/')||path.startsWith('//'))fail(400,'Нужен локальный путь API')
  const url=new URL(path,'https://demo.local');const pathname=url.pathname.replace(/^\/api\/creata\/v1(?=\/|$)/,'');let p:string[];try{p=pathname.split('/').filter(Boolean).map(decodeURIComponent)}catch{fail(400,'Некорректный путь API')}
  const method=(init.method??'GET').toUpperCase();const form=init.body instanceof FormData?init.body:undefined;let body:Body={}
  if(!supportsRoute(method,p.join('/')))fail(404,`Демо не поддерживает ${method} ${pathname}`,'demo_endpoint_unavailable')
  if(init.body&&!form){try{body=JSON.parse(String(init.body));if(!body||Array.isArray(body)||typeof body!=='object')throw Error()}catch{fail(400,'Некорректный JSON запроса')}}
  const prepared=form&&method==='POST'&&p.join('/')==='admin/media'?await prepareMediaUpload(form):undefined
  if(init.signal?.aborted)throw new DOMException('Aborted','AbortError')
  // No awaits may occur between this fresh read and persistence below: an
  // upload's file IO can overlap edits from another engine sharing storage.
  const c=context()
  if(p[0]==='admin')requireRole(c.state,['chief_editor','admin','editor','author'])
  let result=auth(c,method,p,body)
  if(result===undefined&&prepared)result=uploadMedia(c,prepared)
  if(result===undefined)result=editorial(c,method,p,body,url.searchParams)
  if(result===undefined)result=hermes(c,method,p,body)
  if(result===undefined)result=auxiliary(c,method,p,body,url.searchParams)
  if(result===undefined)fail(404,`Демо не поддерживает ${method} ${pathname}`,'demo_endpoint_unavailable')
  if(!['GET','HEAD','OPTIONS'].includes(method))store.write(c.state)
  return clone(result) as T
 }
 function getPublishedArticles():Article[] {return clone(Object.values(context().state.publications).sort((a,b)=>(b.publishedAt??'').localeCompare(a.publishedAt??'')))}
 return {request,getPublishedArticles,getArticle:(idOrSlug:string):Article|undefined=>getPublishedArticles().find(a=>a.id===idOrSlug||a.slug===idOrSlug),getSnapshot:()=>clone(context().state),reset:store.reset}
}
