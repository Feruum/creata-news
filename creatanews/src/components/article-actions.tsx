'use client'
import Link from 'next/link'
import { Bookmark, Check, Link as LinkIcon } from 'lucide-react'
import { useEffect,useState } from 'react'
import type { Article,Locale,Opportunity } from '@/lib/models'
import { copy,localePath } from '@/lib/i18n'
import { useAuth } from './auth-context'
import { completeViewEvent,prepareViewEvent } from '@/lib/view-events'
export function SaveButton({id,type,locale,className='button button-outline',labels}: {id:string;type:'article'|'opportunity';locale:Locale;className?:string;labels?:{save:string;saved:string}}) {
  const {auth,loading,request}=useAuth(), [saved,setSaved]=useState(false),[busy,setBusy]=useState(false),[message,setMessage]=useState(''),t=copy[locale]
  useEffect(()=>{if(!auth.user){setSaved(false);return}let active=true;request<{articles:Article[];opportunities:Opportunity[]}>('/account/saved').then(result=>{if(active)setSaved((type==='article'?result.articles:result.opportunities).some(item=>item.id===id))}).catch(()=>{});return()=>{active=false}},[auth.user,id,type,request])
  async function toggle(){setBusy(true);setMessage('');try{await request(`/account/saved/${type}/${encodeURIComponent(id)}`,{method:saved?'DELETE':'POST'});setSaved(!saved)}catch{setMessage(t.operationFailed)}finally{setBusy(false)}}
  if(!loading&&!auth.user)return <Link href={localePath(locale,`/profile?returnTo=${encodeURIComponent(windowSafePath(locale))}`)} className={className}><Bookmark size={15}/>{labels?.save||t.saveArticle}</Link>
  return <><button className={className} disabled={loading||busy} onClick={toggle} aria-pressed={saved} aria-label={saved?t.removeSaved:t.saveArticle}>{saved?<Check size={15}/>:<Bookmark size={15}/>} {saved?labels?.saved||t.removeSaved:labels?.save||t.saveArticle}</button>{message&&<span className='inline-message error' role='alert'>{message}</span>}</>
}
function windowSafePath(locale:Locale){return typeof window==='undefined'?`/${locale}`:window.location.pathname}
export function ShareButton({locale}: {locale:Locale}) {const[copied,setCopied]=useState(false),[error,setError]=useState(false),t=copy[locale];return <><button className='button button-outline' onClick={async()=>{try{await navigator.clipboard.writeText(window.location.href);setCopied(true);setError(false)}catch{setError(true)}}}>{copied?<Check size={15}/>:<LinkIcon size={15}/>} {copied?t.copied:t.copyLink}</button>{error&&<span className='inline-message error' role='alert'>{t.operationFailed}</span>}</>}
export function ViewEvent({id}: {id:string}) {useEffect(()=>{let storage:Storage|null=null;try{storage=window.sessionStorage}catch{};const event=prepareViewEvent(storage,id);if(event.delivered)return;void fetch('/api/creata/v1/public/events',{method:'POST',headers:{'Content-Type':'application/json'},credentials:'include',body:JSON.stringify({type:'view',targetId:id,eventId:event.eventId})}).then(response=>{if(response.ok)completeViewEvent(storage,id,event.eventId)}).catch(()=>{})},[id]);return null}
export function OpportunityLink({id,url,locale,className='button button-accent',label}: {id:string;url:string;locale:Locale;className?:string;label?:string}) {return <a href={url} target='_blank' rel='noopener noreferrer' className={className} onClick={()=>{void fetch('/api/creata/v1/public/events',{method:'POST',headers:{'Content-Type':'application/json'},credentials:'include',body:JSON.stringify({type:'opportunity_click',targetId:id,eventId:crypto.randomUUID()})}).catch(()=>{})}}>{label||copy[locale].apply}{!label&&' ↗'}</a>}
