type EventIdentity={eventId:string;delivered:boolean}
type TabStorage=Pick<Storage,'getItem'|'setItem'>
const memory=new Map<string,EventIdentity>()
const keyFor=(articleId:string)=>`creata:view:${articleId}`
export function prepareViewEvent(storage:TabStorage|null,articleId:string,uuid=()=>crypto.randomUUID()):EventIdentity {
  const key=keyFor(articleId)
  try{const raw=storage?.getItem(key);if(raw){const value=JSON.parse(raw);if(typeof value.eventId==='string'&&value.eventId.length<=80&&typeof value.delivered==='boolean'){memory.set(key,value);return value}}}catch{}
  let value=memory.get(key)
  if(!value){value={eventId:uuid(),delivered:false};memory.set(key,value)}
  try{storage?.setItem(key,JSON.stringify(value))}catch{}
  return value
}
export function completeViewEvent(storage:TabStorage|null,articleId:string,eventId:string){const key=keyFor(articleId),current=memory.get(key);if(current?.eventId!==eventId)return;const value={eventId,delivered:true};memory.set(key,value);try{storage?.setItem(key,JSON.stringify(value))}catch{}}
