import { expect,test } from 'bun:test'
import { prepareViewEvent,completeViewEvent } from './view-events'
function storage(){const data=new Map<string,string>();return{getItem:(key:string)=>data.get(key)||null,setItem:(key:string,value:string)=>{data.set(key,value)}}}
test('strict-mode remount and failed send reuse one event identity',()=>{
  const store=storage(),first=prepareViewEvent(store,'story-strict',()=> 'stable-first-id')
  const retry=prepareViewEvent(store,'story-strict',()=> 'must-not-be-used')
  expect(first).toEqual({eventId:'stable-first-id',delivered:false});expect(retry).toEqual(first)
  completeViewEvent(store,'story-strict',first.eventId)
  expect(prepareViewEvent(store,'story-strict',()=> 'also-not-used')).toEqual({eventId:'stable-first-id',delivered:true})
})
test('storage failures do not create new event ids within the same tab',()=>{
  const blocked={getItem:()=>{throw Error('blocked')},setItem:()=>{throw Error('blocked')}}
  expect(prepareViewEvent(blocked,'blocked-story',()=> 'fallback-id').eventId).toBe('fallback-id')
  expect(prepareViewEvent(blocked,'blocked-story',()=> 'wrong-id').eventId).toBe('fallback-id')
})
