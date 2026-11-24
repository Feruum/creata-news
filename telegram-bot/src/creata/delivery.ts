import {InlineKeyboard,type Bot} from 'grammy'
import type {Delivery} from '@creata/contracts'
import type {ServiceAPI} from './api'
import {deliveryFailure,splitMessage} from './digest'

export async function deliverOne(bot:Bot,api:ServiceAPI,workerId='creata-telegram',onlyChannel=''){
 const{delivery}=await api.request<{delivery:Delivery|null}>('/internal/deliveries/claim',{workerId})
 if(!delivery)return {status:'idle' as const}
 if(onlyChannel&&delivery.chatId!==onlyChannel)throw new Error('Delivery is outside the configured Telegram channel')
 let sent=0,lastMessage=0
 try{
  const keyboard=new InlineKeyboard();for(const link of delivery.links||[])keyboard.url(link.label,link.url).row()
  const parts=splitMessage(delivery.text)
  for(let i=0;i<parts.length;i++){const result=await bot.api.sendMessage(delivery.chatId,parts[i],{link_preview_options:{is_disabled:true},...(i===parts.length-1&&delivery.links?.length?{reply_markup:keyboard}:{})});sent++;lastMessage=result.message_id}
  await api.request('/internal/deliveries/'+delivery.id+'/complete',{leaseToken:delivery.leaseToken,messageId:String(lastMessage)})
  return {status:'sent' as const,id:delivery.id}
 }catch(error){await api.request('/internal/deliveries/'+delivery.id+'/fail',{leaseToken:delivery.leaseToken,...deliveryFailure(error,sent)});return {status:'failed' as const,id:delivery.id}}
}
