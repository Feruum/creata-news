import { ApiError, type Article, type Author, type AuthState, type Category, type Digest, type Job, type Launch, type MediaAsset, type NewspaperIssue, type NewspaperPublication, type Opportunity, type Settings, type SitePage, type Story, type StoryFollowUp, type User } from '@creata/contracts'

export const DEMO_PIPELINE_KEY = 'creata:pipeline-demo:v1'
export const DEMO_PIPELINE_EVENT = 'creata:pipeline-demo:change'
export interface DemoSeed { articles: Article[]; categories: Category[]; opportunities: Opportunity[]; authors?: Author[]; launches?: Launch[]; pages?: SitePage[]; newspapers?: Array<NewspaperIssue | NewspaperPublication> }
export type DemoStorage = Pick<Storage,'getItem'|'setItem'> & Partial<Pick<Storage,'removeItem'>> | Map<string,string>
export type DemoClock = () => number | Date
export type Body = Record<string, any>
export interface DemoAudit { id:string;actorId:string;action:string;targetId:string;createdAt:string;detail?:unknown }
export interface DemoDelivery { id:string;digestId:string;chatId:string;status:string;attempts:number;createdAt:string;simulated:true;error?:string;messageId?:number }
export interface DemoStory extends Story {updatedAt?:string;mergedInto?:string;mergedStoryIds?:string[]}
export interface DemoMerge {id:string;leftId:string;rightId:string;status:string;createdAt:string;leftBefore:DemoStory;rightBefore:DemoStory;afterLeftVersion?:string;afterRightVersion?:string}
export interface DemoArticle extends Article {storySlug?:string}
export interface FollowedStoryState {
 followedStoryIds:string[]
 seenStoryUpdateIds:Record<string,string[]>
}
export interface DemoState {
 followedStories:Record<string,string[]>
 seenStoryUpdateIds:Record<string,FollowedStoryState['seenStoryUpdateIds']>
 storyFollowUps:Record<string,Pick<StoryFollowUp,'assigneeId'|'dueDate'|'updatedAt'>>
 version:1; articles:DemoArticle[];publications:Record<string,Article>;categories:Category[];authors:Author[];opportunities:Opportunity[];launches:Launch[];pages:SitePage[];pagePublications:Record<string,SitePage>;pageRevisions:Record<string,SitePage[]>;newspapers:NewspaperIssue[];newspaperPublications:Record<string,NewspaperPublication>;media:MediaAsset[];sources:import('@creata/contracts').Source[];stories:DemoStory[];jobs:Job[];settings:Settings;session:AuthState;users:User[];digests:Digest[];deliveries:DemoDelivery[];audit:DemoAudit[];revisions:Record<string,DemoArticle[]>;merges:DemoMerge[];deleted:string[];sequence:number;saved:{articles:string[];opportunities:string[]}
}
export interface Context {state:DemoState;time:number;iso:string;id:(kind:string)=>string;audit:(action:string,id:string,detail?:unknown)=>void}
export const clone = <T>(value:T):T => structuredClone(value)
export const localized=(ru='',kk=ru,en=ru)=>({ru,kk,en})
export function fail(status:number,message:string,code?:string):never { throw new ApiError(status,message,code) }
export function find<T extends {id:string}>(items:T[],id:string):T {return items.find(x=>x.id===id)??fail(404,'Объект не найден в локальной демоверсии','not_found')}
export function expected(item:{revision:number},body:Body) {if(!Number.isInteger(body.expectedRevision)||body.expectedRevision!==item.revision)fail(409,'Материал уже изменён. Обновите страницу и повторите действие.','revision_conflict')}
export function requireRole(state:DemoState,roles:string[]) {if(!state.session.user)fail(401,'Войдите в демоверсию','unauthorized');if(!state.session.user.roles.some(r=>roles.includes(r)))fail(403,'Для этого действия нужна другая редакционная роль','forbidden')}
export function canPublish(user:User,article:Article) {return user.roles.includes('chief_editor')||article.layer==='world'&&user.roles.includes('editor')&&user.verticals.includes(article.categoryId)}
export function validUrl(value:unknown) {try {const url=new URL(String(value));if(!['https:','http:'].includes(url.protocol)||url.username||url.password)throw Error();return url.href}catch{fail(400,'Укажите корректную ссылку HTTP или HTTPS','invalid_url')}}
export function pick(body:Body,keys:string[]) {return Object.fromEntries(keys.filter(k=>body[k]!==undefined).map(k=>[k,clone(body[k])]))}
export function list<T>(items:T[]) {return {items,total:items.length}}
export function makeContext(state:DemoState,time:number):Context {const iso=new Date(time).toISOString();const id=(kind:string)=>`demo-${kind}-${++state.sequence}`;return {state,time,iso,id,audit:(action,targetId,detail)=>{state.audit.unshift({id:id('audit'),actorId:state.session.user?.id||'demo-hermes',action,targetId,createdAt:iso,...(detail===undefined?{}:{detail})})}}}
