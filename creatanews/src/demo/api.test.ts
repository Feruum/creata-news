import {describe,expect,test} from 'bun:test'
import {readDemoPublic} from './api'
import {createDemoRequest} from './browser'
function storage(){const data=new Map<string,string>();return{getItem:(key:string)=>data.get(key)??null,setItem:(key:string,value:string)=>{data.set(key,value)}}}
describe('explicit public demo data',()=>{
 test('lists only published translated samples and applies filters',()=>{
  const all=readDemoPublic('/public/articles?locale=kk') as any
  expect(all.items.length).toBeGreaterThan(5)
  expect(all.items.every((article:any)=>article.status==='published'&&article.title.kk)).toBe(true)
  const graphics=readDemoPublic('/public/articles?format=graphic') as any
  expect(graphics.items).toHaveLength(1)
  expect(graphics.items[0].graphic.verified).toBe(false)
  const filtered=readDemoPublic('/public/articles?category=ai') as any
  expect(filtered.items.every((article:any)=>article.categoryId==='ai')).toBe(true)
  expect((readDemoPublic('/public/articles?q=does-not-exist-xyz') as any).items).toHaveLength(0)
 })
 test('keeps page/media URLs portable and protects unpublished/admin resources',()=>{
  const issue=readDemoPublic('/public/newspapers/01') as any
  expect(issue.pages).toBe(36);expect(issue.pdf).toStartWith('/demo-media/')
  expect(JSON.stringify(readDemoPublic('/public/pages/ecosystem'))).not.toMatch(/localhost|127\.0\.0\.1/)
  expect(()=>readDemoPublic('/admin/articles')).toThrow()
  expect(()=>readDemoPublic('/public/articles/missing')).toThrow()
 })
})
describe('visitor-local demo actions',()=>{
 test('saves, reader preferences, votes and comments are isolated by browser',async()=>{
  const first=createDemoRequest(storage()),second=createDemoRequest(storage())
  const article=(readDemoPublic('/public/articles') as any).items[0]
  await first('/account/saved/article/'+article.id,{method:'POST'})
  expect((await first<any>('/account/saved')).articles).toHaveLength(1)
  expect((await second<any>('/account/saved')).articles).toHaveLength(0)
  await first('/account',{method:'PATCH',body:JSON.stringify({name:'Demo visitor',locale:'kk',stage:'building',topics:['ai'],frequency:'weekly',paused:true,roles:['admin']})})
  const me=await first<any>('/auth/me');expect(me.user.name).toBe('Demo visitor');expect(me.user.roles).toEqual(['reader'])
  expect((await second<any>('/auth/me')).user.name).not.toBe('Demo visitor')
  const launch=(readDemoPublic('/public/launches') as any).items[0]
  const vote=await first<any>('/account/launches/'+launch.id+'/vote',{method:'POST',body:JSON.stringify({voted:true})})
  expect(vote.votes).toBe(launch.votes+1)
  await first('/account/launches/'+launch.id+'/comments',{method:'POST',body:JSON.stringify({body:'Only in this browser',clientId:'same-id'})})
  await first('/account/launches/'+launch.id+'/comments',{method:'POST',body:JSON.stringify({body:'Only in this browser',clientId:'same-id'})})
  expect((await first<any>('/public/launches/'+launch.slug+'/comments')).items).toHaveLength(1)
  expect((await second<any>('/public/launches/'+launch.slug+'/comments')).items).toHaveLength(0)
 })
 test('rejects real delivery/admin operations and supports logout/demo login',async()=>{
  const request=createDemoRequest(storage())
  await expect(request('/account/telegram-link',{method:'POST'})).rejects.toThrow()
  await expect(request('/admin/articles')).rejects.toThrow()
  await request('/auth/logout',{method:'POST'});expect((await request<any>('/auth/me')).user).toBeNull()
  await request('/auth/demo-login',{method:'POST'});expect((await request<any>('/auth/me')).user.roles).toEqual(['reader'])
 })
})
