'use client'
import Link from 'next/link'
import { createContext,useContext,useEffect,useState,type ReactNode } from 'react'
import { Triangle } from 'lucide-react'
import type { LaunchVoteState,List,Locale } from '@creata/contracts'
import { useAuth } from '@/components/auth-context'
import { AnimatedNumber } from '@/components/skiper'
import { copy,localePath } from '@/lib/i18n'
import { launchSocialCopy } from './copy'
import styles from './launches.module.css'

type Votes={ids:Set<string>;ready:boolean;update:(id:string,voted:boolean)=>void}
const VotesContext=createContext<Votes|null>(null)
export function LaunchVotesProvider({locale,children}:{locale:Locale;children:ReactNode}) {
  const {auth,request,loading}=useAuth(),[ids,setIds]=useState<Set<string>>(new Set()),[ready,setReady]=useState(false),[failed,setFailed]=useState(false),[retry,setRetry]=useState(0)
  const userID=auth.user?.id
  useEffect(()=>{
    let active=true
    setIds(new Set());setFailed(false);setReady(false)
    if(!userID){setReady(!loading);return}
    request<List<string>>('/account/launch-votes').then(data=>{if(active){setIds(new Set(data.items));setReady(true)}}).catch(()=>{if(active)setFailed(true)})
    return()=>{active=false}
  },[userID,loading,request,retry])
  return <VotesContext.Provider value={{ids,ready,update:(id,voted)=>setIds(current=>{const next=new Set(current);if(voted)next.add(id);else next.delete(id);return next})}}>{children}{failed&&<p className='inline-message error' role='alert'>{copy[locale].operationFailed} <button type='button' onClick={()=>setRetry(value=>value+1)}>{copy[locale].retry}</button></p>}</VotesContext.Provider>
}

export function LaunchVoteButton({id,slug,name,initialVotes,locale}:{id:string;slug:string;name:string;initialVotes:number;locale:Locale}) {
  const {auth,loading,request,demo}=useAuth(),context=useContext(VotesContext),[count,setCount]=useState(initialVotes),[busy,setBusy]=useState(false),[error,setError]=useState(false)
  const voted=context?.ids.has(id)||false,l=launchSocialCopy[locale]
  useEffect(()=>setCount(initialVotes),[initialVotes])
  useEffect(()=>{if(demo)setCount(initialVotes+(voted?1:0))},[demo,initialVotes,voted])
  const content=<><Triangle size={17} aria-hidden='true'/><AnimatedNumber value={count}/></>
  if(!loading&&!auth.user)return <Link className={styles.counter} aria-label={`${l.voteSignIn}: ${name} (${count})`} href={localePath(locale,`/profile?returnTo=${encodeURIComponent(localePath(locale,`/launches/${slug}`))}`)}>{content}</Link>
  async function toggle(){
    setBusy(true);setError(false)
    try{const result=await request<LaunchVoteState>(`/account/launches/${encodeURIComponent(id)}/vote`,{method:'POST',body:JSON.stringify({voted:!voted})});setCount(result.votes);context?.update(id,result.voted)}catch{setError(true)}finally{setBusy(false)}
  }
  return <span className={styles.voteControl}><button type='button' className={styles.counter} disabled={loading||busy||!context?.ready} aria-pressed={voted} aria-label={`${voted?l.unvote:l.vote}: ${name} (${count})`} onClick={()=>void toggle()}>{content}</button>{error&&<span className={styles.voteError} role='alert'>{copy[locale].operationFailed}</span>}</span>
}
