'use client'
import Link from 'next/link'
import { useCallback,useEffect,useRef,useState } from 'react'
import type { LaunchComment,List,Locale } from '@creata/contracts'
import { useAuth } from '@/components/auth-context'
import { copy,formatDate,localePath } from '@/lib/i18n'
import { launchSocialCopy } from './copy'
import styles from './launches.module.css'

export function LaunchDiscussion({launchId,slug,locale}:{launchId:string;slug:string;locale:Locale}) {
  const {auth,request,loading}=useAuth(),[comments,setComments]=useState<LaunchComment[]>([]),[total,setTotal]=useState(0),[pending,setPending]=useState(true),[busy,setBusy]=useState(false),[failed,setFailed]=useState(false),[body,setBody]=useState('')
  const submission=useRef<{id:string;body:string}|null>(null),t=copy[locale],l=launchSocialCopy[locale]
  const userID=auth.user?.id
  const load=useCallback(async(offset=0)=>{
    const result=await request<List<LaunchComment>>(`/public/launches/${encodeURIComponent(slug)}/comments?offset=${offset}`)
    setComments(current=>offset?[...current,...result.items.filter(item=>!current.some(existing=>existing.id===item.id))]:result.items)
    setTotal(result.total??result.items.length)
  },[request,slug])
  useEffect(()=>{
    let active=true
    setPending(true);setFailed(false)
    request<List<LaunchComment>>(`/public/launches/${encodeURIComponent(slug)}/comments`).then(result=>{if(active){setComments(result.items);setTotal(result.total??result.items.length)}}).catch(()=>{if(active)setFailed(true)}).finally(()=>{if(active)setPending(false)})
    return()=>{active=false}
  },[request,slug,userID])
  async function post(){
    if(!body.trim())return
    setBusy(true);setFailed(false)
    if(!submission.current||submission.current.body!==body)submission.current={id:crypto.randomUUID(),body}
    try{await request(`/account/launches/${encodeURIComponent(launchId)}/comments`,{method:'POST',body:JSON.stringify({body,clientId:submission.current.id})});setBody('');submission.current=null;await load()}catch{setFailed(true)}finally{setBusy(false)}
  }
  async function remove(id:string){
    if(!window.confirm(l.removeConfirm))return
    setBusy(true);setFailed(false)
    try{await request(`/account/launches/${encodeURIComponent(launchId)}/comments/${encodeURIComponent(id)}`,{method:'DELETE'});await load()}catch{setFailed(true)}finally{setBusy(false)}
  }
  return <section id='discussion' className={styles.discussion}>
    <h2>{l.discussion}{!pending&&<span className={styles.commentTotal}> {total}</span>}</h2>
    {auth.user?<form onSubmit={event=>{event.preventDefault();void post()}} className={styles.commentForm}><label htmlFor='launch-comment'>{l.writeComment}</label><textarea id='launch-comment' required maxLength={2000} value={body} disabled={busy} onChange={event=>setBody(event.target.value)} rows={3}/><button className='button button-accent' disabled={busy||!body.trim()}>{busy?t.saving:l.postComment}</button></form>:!loading&&<p><Link className='text-link' href={localePath(locale,`/profile?returnTo=${encodeURIComponent(localePath(locale,`/launches/${slug}`)+'#discussion')}`)}>{l.signInDiscussion}</Link></p>}
    {failed&&<p className='inline-message error' role='alert'>{t.operationFailed} <button type='button' onClick={()=>{setFailed(false);void load().catch(()=>setFailed(true))}}>{t.retry}</button></p>}
    {pending?<p role='status'>{l.loading}</p>:!comments.length?<p className={styles.count}>{l.noComments}</p>:<ol className={styles.comments}>{comments.map(comment=><li key={comment.id}><header><strong>{comment.authorName}</strong><time dateTime={comment.createdAt}>{formatDate(comment.createdAt,locale,true)}</time></header><p>{comment.body}</p>{comment.canDelete&&<button type='button' disabled={busy} className={styles.removeComment} onClick={()=>void remove(comment.id)}>{l.removeComment}</button>}</li>)}</ol>}
    {comments.length<total&&<button className='button button-outline' disabled={busy} onClick={async()=>{setBusy(true);try{await load(comments.length)}catch{setFailed(true)}finally{setBusy(false)}}}>{t.readMore}</button>}
  </section>
}
