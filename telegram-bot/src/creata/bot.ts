import {Bot,InlineKeyboard,type Context,type BotConfig} from 'grammy'
import type {Category,Digest,List,User} from '@creata/contracts'
import {ServiceAPI} from './api'
import {copy} from './copy'
import {deliveryFailure,formatDigest,localeFrom,splitMessage} from './digest'

export function createBot(token:string,api:ServiceAPI,publicURL:string,options?:BotConfig<Context>){
  const bot=new Bot(token,options)
  const user=(ctx:Context)=>api.request<User>('/internal/bot/users/upsert',{telegramId:String(ctx.from!.id),username:ctx.from?.username,locale:localeFrom(ctx.from?.language_code)})
  const update=(ctx:Context,patch:unknown)=>api.request<User>('/internal/bot/users/'+ctx.from!.id,patch,'PATCH')
  const languages=()=>new InlineKeyboard().text('RU','lang:ru').text('ҚАЗ','lang:kk').text('EN','lang:en')
  async function topics(ctx:Context,profile:User){
    const categories=await api.request<List<Category>>('/internal/bot/topics')
    const keyboard=new InlineKeyboard()
    for(const item of categories.items){keyboard.text((profile.topics.includes(item.id)?'✓ ':'')+item.name[profile.locale],'topic:'+item.id).row()}
    keyboard.text(copy[profile.locale].done,'frequency')
    await ctx.reply(copy[profile.locale].topics,{reply_markup:keyboard})
  }
  bot.use(async(ctx,next)=>{if(ctx.chat?.type!=='private'){await ctx.reply('Please open Creata News in a private chat.');return}await next()})
  bot.command('start',async ctx=>{
    let profile=await user(ctx)
    if(ctx.match){const token=ctx.match.trim().replace(/^link_/,'');profile=await api.request<User>('/internal/bot/link',{token,telegramId:String(ctx.from!.id)});await ctx.reply(copy[profile.locale].linked)}
    await ctx.reply(copy[profile.locale].welcome+'\n\n'+copy[profile.locale].privacy,{reply_markup:languages()})
  })
  bot.command('settings',async ctx=>{const profile=await user(ctx);const t=copy[profile.locale];await ctx.reply(t.settings,{reply_markup:new InlineKeyboard().text(t.language,'languages').row().text(t.interests,'topics').row().text(t.frequency,'frequency')})})
  bot.command('pause',async ctx=>{const profile=await update(ctx,{paused:true});await ctx.reply(copy[profile.locale].paused)})
  bot.command('resume',async ctx=>{const profile=await update(ctx,{paused:false});await ctx.reply(copy[profile.locale].resumed)})
  bot.command('help',async ctx=>{const profile=await user(ctx);await ctx.reply(copy[profile.locale].help)})
  bot.command('digest',async ctx=>{
    const profile=await user(ctx)
    const digest=await api.request<Digest>('/internal/bot/digest',{telegramId:String(ctx.from!.id)})
    const parts=splitMessage(formatDigest(digest,publicURL))
    let sent=0,messageId=0
    try{
      for(let i=0;i<parts.length;i++){
        const message=await ctx.reply(parts[i],{link_preview_options:{is_disabled:true},...(i===parts.length-1?{reply_markup:new InlineKeyboard().url(copy[profile.locale].open,publicURL+'/'+profile.locale)}:{})})
        sent++;messageId=message.message_id
      }
      await api.request('/internal/bot/digests/'+digest.id+'/complete',{telegramId:String(ctx.from!.id),messageId:String(messageId)})
    }catch(error){
      const failure=deliveryFailure(error,sent)
      await api.request('/internal/bot/digests/'+digest.id+'/fail',{telegramId:String(ctx.from!.id),...failure})
      throw error
    }
  })
  bot.callbackQuery(/^lang:(ru|kk|en)$/,async ctx=>{await ctx.answerCallbackQuery();const profile=await update(ctx,{locale:ctx.match[1]});await topics(ctx,profile)})
  bot.callbackQuery('languages',async ctx=>{await ctx.answerCallbackQuery();await ctx.reply((await user(ctx)).locale==='kk'?'Тілді таңдаңыз:':'RU / ҚАЗ / EN',{reply_markup:languages()})})
  bot.callbackQuery('topics',async ctx=>{await ctx.answerCallbackQuery();await topics(ctx,await user(ctx))})
  bot.callbackQuery(/^topic:([\w-]+)$/,async ctx=>{
    await ctx.answerCallbackQuery();const profile=await user(ctx);const id=ctx.match[1]
    const next=profile.topics.includes(id)?profile.topics.filter(x=>x!==id):[...profile.topics,id]
    await topics(ctx,await update(ctx,{topics:next}))
  })
  bot.callbackQuery('frequency',async ctx=>{await ctx.answerCallbackQuery();const p=await user(ctx);const t=copy[p.locale];await ctx.reply(t.frequency,{reply_markup:new InlineKeyboard().text(t.daily,'freq:daily').text(t.weekly,'freq:weekly')})})
  bot.callbackQuery(/^freq:(daily|weekly)$/,async ctx=>{await ctx.answerCallbackQuery();const p=await update(ctx,{frequency:ctx.match[1],paused:false});await ctx.reply(copy[p.locale].saved,{reply_markup:new InlineKeyboard().url(copy[p.locale].open,publicURL+'/'+p.locale)})})
  bot.catch(async error=>{console.error('Bot update failed',error.error instanceof Error?error.error.name:'Error');try{await error.ctx.reply(copy[localeFrom(error.ctx.from?.language_code)].failed)}catch{}})
  return bot
}
