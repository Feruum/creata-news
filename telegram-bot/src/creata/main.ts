import {ServiceAPI} from './api'
import {configureWebhook,createBot,telegramAllowedUpdates} from './bot'
import {deliverOne} from './delivery'
import {webhookCallback} from 'grammy'

const env=process.env
const token=env.TELEGRAM_BOT_TOKEN||''
const serviceToken=env.CREATA_BOT_SERVICE_TOKEN||''
const channelOnly=env.CREATA_TELEGRAM_CHANNEL_ONLY==='1'
const channelId=channelOnly?env.CREATA_TELEGRAM_CHANNEL_ID||'':''
if(channelOnly&&!/^-100\d+$/.test(channelId))throw new Error('Channel-only Telegram delivery requires a numeric channel ID')
const configured=!!token&&serviceToken.length>=32
const api=new ServiceAPI(env.CREATA_API_ORIGIN||'http://localhost:5555',serviceToken)
const bot=configured?createBot(token,api,env.CREATA_PUBLIC_URL||'http://localhost:3002',undefined,channelOnly):null
let ready=false,stopping=false
const webhookSecret=env.TELEGRAM_WEBHOOK_SECRET||''
if(env.CREATA_ENV==='production'&&(!configured||webhookSecret.length<32))throw new Error('Production bot credentials and webhook secret are required')
const webhook=bot&&webhookSecret?webhookCallback(bot,'std/http',{secretToken:webhookSecret}):null
const server=Bun.serve({port:Number(env.HTTP_PORT||3001),async fetch(req){
  const path=new URL(req.url).pathname
  if(path==='/health/live')return Response.json({status:'alive'})
  if(path==='/health/ready')return Response.json({ready,configured},{status:ready?200:503})
  if(path==='/telegram/webhook'&&webhook&&req.method==='POST')return webhook(req)
  return Response.json({error:'Not found'},{status:404})
}})

async function deliver(){
  if(!bot)return
  while(!stopping){
    try{
      if((await deliverOne(bot,api,undefined,channelId)).status==='idle')await Bun.sleep(2000)
    }catch{console.error('Delivery queue unavailable');await Bun.sleep(5000)}
  }
}
if(bot){
  await bot.init()
  if(channelOnly){
    const chat=await bot.api.getChat(channelId)
    const member=await bot.api.getChatMember(channelId,bot.botInfo.id)
    if(chat.type!=='channel'||member.status!=='administrator'||!member.can_post_messages)throw new Error('The bot must be a channel administrator with permission to post')
  }else{
    await bot.api.setMyCommands([{command:'start',description:'Creata News'},{command:'digest',description:'RU / ҚАЗ / EN — digest'},{command:'settings',description:'Digest settings'},{command:'pause',description:'Pause delivery'},{command:'resume',description:'Resume delivery'},{command:'help',description:'Help'}])
  }
  if(env.TELEGRAM_WEBHOOK_URL){
    if(webhookSecret.length<32)throw new Error('Webhook secret must contain at least 32 characters')
    await configureWebhook(bot,env.TELEGRAM_WEBHOOK_URL,webhookSecret,channelOnly)
    ready=true
  }else{
    if(env.CREATA_ENV==='production')throw new Error('Production webhook URL is required')
    void bot.start({allowed_updates:telegramAllowedUpdates(channelOnly),onStart:()=>{ready=true}})
  }
  void deliver()
}else console.log('Creata Telegram: credentials not configured; health endpoint remains available')
async function stop(){stopping=true;ready=false;if(bot?.isRunning())await bot.stop();await server.stop(true)}
process.on('SIGINT',()=>void stop());process.on('SIGTERM',()=>void stop())
