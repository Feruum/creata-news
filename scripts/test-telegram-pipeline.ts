import assert from 'node:assert/strict'
import {createBot} from '../telegram-bot/src/creata/bot'
import {ServiceAPI} from '../telegram-bot/src/creata/api'
import {deliverOne} from '../telegram-bot/src/creata/delivery'

const origin=process.env.CREATA_TEST_API_ORIGIN||'',url=new URL(origin)
if(process.env.CREATA_ENV!=='test'||!['localhost','127.0.0.1','::1'].includes(url.hostname))throw Error('Telegram fixture requires the isolated local test API')
const service=new ServiceAPI(origin,process.env.CREATA_BOT_SERVICE_TOKEN||''),chat=900000000+Date.now()%99000000
let messageId=0,rateLimit=false,rejected=0
const sends:Array<{chat:string;text:string}>=[]
const transport=Bun.serve({hostname:'127.0.0.1',port:0,async fetch(request){
 const method=new URL(request.url).pathname.split('/').pop()
 if(method==='getMe')return Response.json({ok:true,result:{id:12345,is_bot:true,first_name:'Creata fixture',username:'creata_fixture_bot'}})
 const body=await request.json() as {chat_id?:number|string;text?:string}
 if(method==='sendMessage'){
  if(rateLimit){rateLimit=false;rejected++;return Response.json({ok:false,error_code:429,description:'Too Many Requests',parameters:{retry_after:1}},{status:429})}
  sends.push({chat:String(body.chat_id),text:body.text||''})
  return Response.json({ok:true,result:{message_id:++messageId,date:Math.floor(Date.now()/1000),chat:{id:body.chat_id,type:String(body.chat_id).startsWith('-')?'channel':'private'},text:body.text}})
 }
 return Response.json({ok:true,result:true})
}})
const bot=createBot('12345:local-fixture-token',service,process.env.CREATA_PUBLIC_URL||'http://localhost:3002',{client:{apiRoot:`http://127.0.0.1:${transport.port}`}})
let cookie='',csrf='',digestId=''
async function admin(path:string,data?:unknown,method=data===undefined?'GET':'POST'){
 const res=await fetch(origin+'/api/creata/v1'+path,{method,headers:{'Content-Type':'application/json',Cookie:cookie,'X-CSRF-Token':csrf},body:data===undefined?undefined:JSON.stringify(data)})
 assert.equal(res.ok,true,`${path}: ${res.status} ${res.ok?'':await res.text()}`)
 const set=res.headers.getSetCookie?.()[0];if(set)cookie=set.split(';')[0]
 return res.json() as Promise<any>
}
async function command(text:string){await bot.handleUpdate({update_id:++messageId,message:{message_id:messageId,date:Math.floor(Date.now()/1000),from:{id:chat,is_bot:false,first_name:'Local fixture',language_code:'ru'},chat:{id:chat,type:'private',first_name:'Local fixture'},text,entities:[{type:'bot_command',offset:0,length:text.length}]}})}
try{
 await bot.init()
 await command('/start');assert.ok(sends.some(m=>m.chat===String(chat)&&m.text.includes('CR≡ATA News')))
 await command('/pause');let user=await service.request<any>('/internal/bot/users/upsert',{telegramId:String(chat),locale:'ru'});assert.equal(user.paused,true)
 await command('/resume');user=await service.request<any>('/internal/bot/users/upsert',{telegramId:String(chat),locale:'ru'});assert.equal(user.paused,false)
 await command('/digest');assert.ok(sends.some(m=>m.chat===String(chat)&&m.text.includes('/ru/articles/')))
 await command('/pause')
 await admin('/auth/dev-login',{role:'admin'});csrf=(await admin('/auth/me')).csrfToken
 const digest=await admin('/admin/digests/preview',{locale:'ru',kind:'channel'});digestId=digest.id
 await admin(`/admin/digests/${digest.id}/approve`,{})
 await admin(`/admin/digests/${digest.id}/send`,{})
 const deliveries=(await admin('/admin/deliveries')).items as any[],target=deliveries.find(d=>d.digestId===digest.id)
 assert.ok(target,'Channel delivery was not queued')
 rateLimit=true
 const first=await deliverOne(bot,service,'telegram-http-fixture');assert.equal(first.status,'failed');assert.equal(rejected,1)
 await Bun.sleep(1200)
 let delivered=false
 for(let attempt=0;attempt<40;attempt++){
  await deliverOne(bot,service,'telegram-http-fixture')
  const all=(await admin('/admin/deliveries')).items as any[]
  if(all.find(d=>d.id===target.id)?.status==='sent'){delivered=true;break}
  await Bun.sleep(1100)
 }
 assert.equal(delivered,true,'Telegram 429 retry did not confirm channel delivery')
 const before=sends.filter(m=>m.chat===target.chatId).length
 await deliverOne(bot,service,'telegram-http-fixture')
 assert.equal(sends.filter(m=>m.chat===target.chatId).length,before,'Confirmed channel delivery repeated')
 console.log('Telegram HTTP fixture: real bot onboarding, pause/resume, personal digest, channel digest, 429 retry and no duplicate confirmation passed')
}finally{
 try{await service.request('/internal/bot/users/'+chat,{paused:true},'PATCH')}catch{}
 if(digestId)console.log('Verified fixture digest',digestId)
 await transport.stop(true)
}
