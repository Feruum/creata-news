import type { Article, StoryFollowUp } from '@creata/contracts'
import { canPublish, fail, list, requireRole, type Body, type Context, type DemoState } from './model'

type Schedule = DemoState['storyFollowUps'][string]

function storedReminder(state:DemoState,id:string) {
 return Object.hasOwn(state.storyFollowUps,id)?state.storyFollowUps[id]:undefined
}

function reminderItem(state:DemoState,article:Article,schedule:Schedule):StoryFollowUp {
 const {assigneeId,dueDate,updatedAt}=schedule
 const assigneeName=state.users.find(user=>user.id===assigneeId)?.name??''
 const {ru,kk,en}=article.title??{}
 const title={ru:typeof ru==='string'?ru:'',kk:typeof kk==='string'?kk:'',en:typeof en==='string'?en:''}
 return {articleId:article.id,title,assigneeId,assigneeName,dueDate,updatedAt}
}

export function storyFollowUpQueue(c:Context) {
 const {state}=c
 requireRole(state,['chief_editor','editor'])
 const user=state.session.user!
 const items:StoryFollowUp[]=[]
 for(const article of state.articles){
  const reminder=storedReminder(state,article.id)
  if(reminder&&canPublish(user,article))items.push(reminderItem(state,article,reminder))
 }
 return list(items.sort((a,b)=>a.dueDate.localeCompare(b.dueDate)||a.articleId.localeCompare(b.articleId)))
}

function validDueDate(value:unknown):value is string {
 if(typeof value!=='string'||!/^\d{4}-\d{2}-\d{2}$/.test(value)||value.startsWith('0000-'))return false
 const date=new Date(`${value}T00:00:00Z`)
 return Number.isFinite(date.getTime())&&date.toISOString().slice(0,10)===value
}

export function storyFollowUp(c:Context,article:Article,method:string,body:Body) {
 const {state}=c
 if(!canPublish(state.session.user!,article))fail(403,'Недостаточно прав для планирования продолжения сюжета')
 const reminder=storedReminder(state,article.id)
 const eligible=Boolean(article.storyUpdates?.length||state.publications[article.id]?.storyUpdates?.length)
 if(method==='GET')return {item:reminder?reminderItem(state,article,reminder):null,eligible}
 if(method==='DELETE'){
  if(reminder){
   delete state.storyFollowUps[article.id]
   const {dueDate,assigneeId}=reminder
   c.audit('story_follow_up.complete',article.id,{dueDate,assigneeId})
  }
  return {ok:true}
 }
 const {dueDate,assigneeId}=body
 if(!eligible)fail(400,'Добавьте обновление сюжета перед планированием продолжения')
 if(!validDueDate(dueDate))fail(400,'Укажите корректную дату продолжения сюжета')
 if(Object.keys(body).length!==2||typeof assigneeId!=='string'||!assigneeId||assigneeId.trim()!==assigneeId||assigneeId.includes('\0')){
  fail(400,'Укажите корректные данные редактора')
 }
 const assignee=state.users.find(user=>user.id===assigneeId)
 if(!assignee||!canPublish(assignee,article))fail(400,'Назначьте редактора, который может опубликовать материал')
 const schedule={dueDate,assigneeId:assignee.id,updatedAt:c.iso}
 state.storyFollowUps={...state.storyFollowUps,[article.id]:schedule}
 c.audit('story_follow_up.set',article.id,{dueDate,assigneeId:assignee.id})
 return {item:reminderItem(state,article,schedule)}
}
