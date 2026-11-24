import {ApiError,type Article,type Launch,type LaunchComment,type Opportunity,type User} from '@creata/contracts'
import source from './reader-data.json'
import {createPipelineDemo,followedStoryRequest,type DemoSeed,type FollowedStoryState} from '@creata/demo-pipeline'
const seed=source as unknown as {articles:Article[];opportunities:Opportunity[];launches:Launch[];categories:{id:string}[]}
type Store=Pick<Storage,'getItem'|'setItem'>
type Comment=LaunchComment&{clientId:string}
type State=FollowedStoryState&{user:User;loggedOut:boolean;articles:string[];opportunities:string[];votes:string[];comments:Comment[]}
const key='creata:demo:v1',memory=new Map<string,string>()
const fallback:Store={getItem:key=>memory.get(key)??null,setItem:(key,value)=>{memory.set(key,value)}}
function browserStore():Store{try{const storage=window.localStorage;storage.getItem(key);return storage}catch{return fallback}}
function initial():State{return{user:{id:'demo-reader',name:'Creata demo reader',locale:'ru',stage:'exploring',topics:['ai','creative'],frequency:'daily',paused:false,roles:['reader'],verticals:[],telegramLinked:false},loggedOut:false,articles:[],opportunities:[],followedStoryIds:[],seenStoryUpdateIds:{},votes:[],comments:[]}}
function unavailable(status=404,message='Demo resource not found'):never{throw new ApiError(status,message,'demo_only')}
export function createDemoRequest(storage:Store=browserStore()){
 const pipeline=createPipelineDemo(source as unknown as DemoSeed,storage)
 return async function request<T=unknown>(path:string,init:RequestInit={}):Promise<T>{
  const snapshot=pipeline.getSnapshot(),data={...seed,articles:pipeline.getPublishedArticles(),opportunities:snapshot.opportunities,launches:snapshot.launches}
  let state=initial();try{const saved=JSON.parse(storage.getItem(key)||'null');if(saved?.user&&Array.isArray(saved.articles)&&Array.isArray(saved.opportunities)&&Array.isArray(saved.votes)&&Array.isArray(saved.comments))state={...state,...saved,followedStoryIds:Array.isArray(saved.followedStoryIds)?saved.followedStoryIds:[]}}catch{}
  state.user={...state.user,roles:['reader'],verticals:[],telegramLinked:false}
  const url=new URL(path,'https://creata.demo'),parts=url.pathname.split('/').filter(Boolean),method=(init.method||'GET').toUpperCase()
  let body
  try{body=typeof init.body==='string'?JSON.parse(init.body):{}}catch{return unavailable(400,'Invalid JSON request')}
  const save=()=>storage.setItem(key,JSON.stringify(state))
  let result:unknown
  if(url.pathname==='/auth/me')result={user:state.loggedOut?null:state.user,providers:{google:false,telegram:false},devAuth:false}
  else if(url.pathname==='/auth/logout'&&method==='POST'){state.loggedOut=true;save();result={ok:true}}
  else if(url.pathname==='/auth/demo-login'&&method==='POST'){state.loggedOut=false;save();result={ok:true}}
  else if(parts[0]==='public'&&parts[1]==='launches'&&parts[3]==='comments'){
   const launch=data.launches.find(item=>item.slug===parts[2]);if(!launch)return unavailable()
   const items=state.comments.filter(item=>item.launchId===launch.id),offset=Math.max(0,Number(url.searchParams.get('offset'))||0)
   result={items:items.slice(offset,offset+20),total:items.length}
  }else{
   if(parts[0]!=='account')return unavailable()
   if(state.loggedOut)return unavailable(401,'Open the demo profile first')
   if(url.pathname==='/account'&&method==='GET')result=state.user
   else if(url.pathname==='/account'&&method==='PATCH'){
    if(body.name!==undefined&&(typeof body.name!=='string'||!body.name.trim()||body.name.length>120))return unavailable(400,'Enter a name')
    if(body.name!==undefined)state.user.name=body.name.trim()
    if(['ru','kk','en'].includes(body.locale))state.user.locale=body.locale
    if(['exploring','trying','building'].includes(body.stage))state.user.stage=body.stage
    if(['daily','weekly'].includes(body.frequency))state.user.frequency=body.frequency
    if(typeof body.paused==='boolean')state.user.paused=body.paused
    if(Array.isArray(body.topics))state.user.topics=body.topics.filter((id:unknown)=>data.categories.some(category=>category.id===id))
    save();result=state.user
   }else if(url.pathname==='/account/saved'&&method==='GET')result={articles:data.articles.filter(item=>state.articles.includes(item.id)),opportunities:data.opportunities.filter(item=>state.opportunities.includes(item.id))}
   else if(parts[1]==='saved'&&parts.length===4&&['POST','DELETE'].includes(method)){
    const type=parts[2],id=decodeURIComponent(parts[3]);if(type!=='article'&&type!=='opportunity')return unavailable()
    const field=type==='article'?'articles':'opportunities';if(!data[field].some(item=>item.id===id))return unavailable()
    state[field]=method==='POST'?[...new Set([...state[field],id])]:state[field].filter(item=>item!==id);save();result={ok:true}
   }else if(parts[1]==='followed-stories'){
    result=followedStoryRequest(data.articles,state,method,parts.slice(2).map(decodeURIComponent),body)
    if(method!=='GET')save()
   }else if(url.pathname==='/account/launch-votes')result={items:state.votes}
   else if(parts[1]==='launches'){
    const launch=data.launches.find(item=>item.id===parts[2]);if(!launch)return unavailable()
    if(parts[3]==='vote'&&method==='POST'){
     state.votes=body.voted?[...new Set([...state.votes,launch.id])]:state.votes.filter(id=>id!==launch.id);save()
     result={voted:state.votes.includes(launch.id),votes:(launch.votes||0)+(state.votes.includes(launch.id)?1:0)}
    }else if(parts[3]==='comments'&&method==='POST'){
     if(typeof body.body!=='string'||!body.body.trim()||body.body.length>2000||typeof body.clientId!=='string')return unavailable(400,'Enter a comment')
     const existing=state.comments.find(item=>item.launchId===launch.id&&item.clientId===body.clientId)
     const comment=existing||{id:crypto.randomUUID(),launchId:launch.id,authorName:state.user.name,body:body.body.trim(),createdAt:new Date().toISOString(),canDelete:true,clientId:body.clientId}
     if(!existing)state.comments.push(comment);save();result=comment
    }else if(parts[3]==='comments'&&parts[4]&&method==='DELETE'){state.comments=state.comments.filter(item=>item.id!==parts[4]||item.launchId!==launch.id);save();result={ok:true}}
    else return unavailable()
   }else return unavailable(403,'External services are disabled in the frontend demo')
  }
  return structuredClone(result) as T
 }
}
