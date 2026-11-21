import type {Digest,Locale} from '@creata/contracts'
export function splitMessage(text:string):string[]{
  const result:string[]=[]
  while(text.length>3800){
    let boundary=text.lastIndexOf('\n',3800)
    if(boundary<2000)boundary=3800;else boundary++
    if(/[\uD800-\uDBFF]/.test(text[boundary-1]))boundary--
    result.push(text.slice(0,boundary));text=text.slice(boundary)
  }
  if(text)result.push(text)
  return result
}
export function localeFrom(value?:string):Locale{
  const language=value?.split('-')[0]
  return language==='ru'||language==='kk'?language:'en'
}
export function formatDigest(digest:Digest,origin:string):string{
  const locale=localeFrom(digest.locale);const base=origin.replace(/\/$/,'')+'/'+locale
  const headings={ru:['Утро Creata','Возможности','За пределами ваших тем','Новых материалов пока нет.'],kk:['Creata таңы','Мүмкіндіктер','Қызығушылықтарыңыздан тыс','Әзірге жаңа материалдар жоқ.'],en:['Creata Morning','Opportunities','Outside your interests','No new stories yet.']}[locale]
  const lines=[headings[0],'']
  for(const article of digest.articles||[])lines.push(article.title[locale],base+'/articles/'+encodeURIComponent(article.slug),'')
  if(digest.opportunities?.length){lines.push(headings[1],'');for(const item of digest.opportunities)lines.push(item.title[locale],base+'/opportunities/'+encodeURIComponent(item.id),'')}
  if(digest.outsideInterest)lines.push(headings[2],digest.outsideInterest.title[locale],base+'/articles/'+encodeURIComponent(digest.outsideInterest.slug))
  if(!(digest.articles?.length||digest.opportunities?.length||digest.outsideInterest))lines.push(headings[3])
  return lines.join('\n')
}
export function deliveryFailure(error:unknown,sent:number):{error:string;retryAfterSeconds?:number;unknown:boolean}{
  const e=error as {error_code?:number;parameters?:{retry_after?:number};description?:string}
  if(e.error_code===429)return {error:'Telegram rate limit',retryAfterSeconds:Math.max(1,e.parameters?.retry_after||60),unknown:sent>0}
  if(e.error_code)return {error:e.description||'Telegram delivery rejected',unknown:sent>0}
  return {error:'Telegram delivery outcome unknown',unknown:true}
}
