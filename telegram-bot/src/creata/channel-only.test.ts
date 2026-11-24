import {expect,spyOn,test} from 'bun:test'
import {ServiceAPI} from './api'
import {configureWebhook,createBot} from './bot'
import {deliverOne} from './delivery'

const botInfo={
 id:12345,is_bot:true as const,first_name:'Pilot',username:'pilot_test_bot',
 can_join_groups:false,can_read_all_group_messages:false,supports_inline_queries:false,
 can_connect_to_business:false,has_main_web_app:false,has_topics_enabled:false,
 allows_users_to_create_topics:false,can_manage_bots:false,supports_join_request_queries:false,
}

test('normal webhook startup clears the channel-only update filter',async()=>{
 const bot=createBot('12345:pilot-test-token',new ServiceAPI('http://localhost','test-token'),'http://localhost',{botInfo})
 const setWebhook=spyOn(bot.api,'setWebhook').mockResolvedValue(true)
 const url='https://example.org/telegram/webhook',secret='test-webhook-secret'
 await configureWebhook(bot,url,secret,true)
 await configureWebhook(bot,url,secret,false)
 expect(setWebhook.mock.calls).toEqual([
  [url,{secret_token:secret,allowed_updates:['channel_post']}],
  [url,{secret_token:secret,allowed_updates:[]}],
 ])
})

test('the channel-only bot ignores commands in private chats',async()=>{
 class PrivateBotAPI extends ServiceAPI {
  requests=0
  override async request<T>():Promise<T>{this.requests++;throw new Error('Unexpected private request')}
 }
 const api=new PrivateBotAPI('http://localhost','test-token')
 const bot=createBot('12345:pivot-test-token',api,'http://localhost',{botInfo},true)
 let telegramCalls=0
 bot.api.config.use(async()=>{telegramCalls++;throw new Error('Unexpected Telegram request')})
 await bot.handleUpdate({update_id:1,message:{
  message_id:1,date:0,text:'/start',chat:{id:20,type:'private',first_name:'Reader'},
  from:{id:20,is_bot:false,first_name:'Reader'},entities:[{type:'bot_command',offset:0,length:6}],
 }})
 expect(api.requests).toBe(0)
 expect(telegramCalls).toBe(0)
})

test('a delivery addressed to another chat never reaches Telegram',async()=>{
 class WrongChatAPI extends ServiceAPI {
  override async request<T>(path:string):Promise<T>{
   if(path==='/internal/deliveries/claim'){
    return {delivery:{id:'outside',chatId:'20',text:'Unexpected',leaseToken:'test-lease'}} as T
   }
   throw new Error('Unexpected delivery update: '+path)
  }
 }
 const api=new WrongChatAPI('http://localhost','test-token')
 const bot=createBot('12345:pivot-test-token',api,'http://localhost',{botInfo},true)
 let telegramCalls=0
 bot.api.config.use(async()=>{telegramCalls++;throw new Error('Unexpected Telegram request')})
 await expect(deliverOne(bot,api,'pilot','-100123456')).rejects.toThrow('Delivery is outside the configured Telegram channel')
 expect(telegramCalls).toBe(0)
})
