/** Pool repeated HTML/data URLs so immutable snapshots fit browser storage. */
export function encodeState(value:unknown):string {
 const pool:string[]=[];const indices=new Map<string,number>()
 const encode=(item:unknown):unknown=>{
  if(typeof item==='string'&&item.length>8192){let index=indices.get(item);if(index===undefined){index=pool.length;indices.set(item,index);pool.push(item)}return {__creata_demo_text__:index}}
  if(Array.isArray(item))return item.map(encode)
  if(item&&typeof item==='object')return Object.fromEntries(Object.entries(item).map(([key,entry])=>[key,encode(entry)]))
  return item
 }
 return JSON.stringify({...encode(value) as object,_demoTextPool:pool})
}
export function decodeState(raw:string):any {
 const data=JSON.parse(raw);const pool=data._demoTextPool
 if(!Array.isArray(pool))return data
 const decode=(item:any):any=>{
  if(Array.isArray(item))return item.map(decode)
  if(item&&typeof item==='object'){
   if(Object.keys(item).length===1&&Number.isInteger(item.__creata_demo_text__)&&typeof pool[item.__creata_demo_text__]==='string')return pool[item.__creata_demo_text__]
   return Object.fromEntries(Object.entries(item).map(([key,entry])=>[key,decode(entry)]))
  }
  return item
 }
 delete data._demoTextPool;return decode(data)
}
