'use client'
import { useCallback,useEffect,useRef,useState } from 'react'
import type { Locale,Opportunity } from '@/lib/models'
import { localize } from '@/lib/i18n'
import { demoMode } from '@/demo/config'
import { getDemoPipeline } from '@/demo/pipeline'
export function useLiveOpportunities(initial:Opportunity[],locale:Locale,initialError:boolean){
  const[items,setItems]=useState(initial),[error,setError]=useState(initialError),active=useRef<AbortController|null>(null)
  useEffect(()=>{setItems(initial);setError(initialError)},[initial,initialError,locale])
  const refresh=useCallback(async()=>{
    if(active.current||document.visibilityState==='hidden')return
    const controller=new AbortController();active.current=controller
    try{
      if(demoMode){
        const engine=await getDemoPipeline()
        if(!controller.signal.aborted){setItems(engine.getSnapshot().opportunities.filter(item=>item.status==='approved'&&localize(item.title,locale)));setError(false)}
        return
      }
      const response=await fetch(`/api/creata/v1/public/opportunities?locale=${locale}`,{cache:'no-store',signal:AbortSignal.any([controller.signal,AbortSignal.timeout(10000)])})
      if(!response.ok)throw new Error('Opportunity refresh failed')
      const data=await response.json()
      if(!Array.isArray(data.items))throw new Error('Invalid opportunity response')
      if(!controller.signal.aborted){setItems(data.items.filter((item:Opportunity)=>item.status==='approved'&&localize(item.title,locale)));setError(false)}
    }catch{if(!controller.signal.aborted)setError(true)}
    finally{if(active.current===controller)active.current=null}
  },[locale])
  useEffect(()=>{
    const onVisible=()=>{if(document.visibilityState==='visible')void refresh()}
    const timer=setInterval(()=>void refresh(),60000)
    window.addEventListener('focus',onVisible);document.addEventListener('visibilitychange',onVisible)
    if(demoMode){void refresh();window.addEventListener('creata:pipeline-demo:change',onVisible);window.addEventListener('storage',onVisible)}
    return()=>{clearInterval(timer);window.removeEventListener('focus',onVisible);document.removeEventListener('visibilitychange',onVisible);window.removeEventListener('creata:pipeline-demo:change',onVisible);window.removeEventListener('storage',onVisible);active.current?.abort();active.current=null}
  },[refresh])
  useEffect(()=>{if(initialError)void refresh()},[initialError,refresh])
  return {items,error,refresh}
}
