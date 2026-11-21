'use client'
import {useEffect,useState} from 'react'
import type {Article} from '@/lib/models'
import type {DemoState} from '@creata/demo-pipeline'
import {demoMode} from './config'
import {getDemoPipeline} from './pipeline'
export function useDemoArticles(initial:Article[]){
 const[articles,setArticles]=useState(initial)
 useEffect(()=>{
  if(!demoMode)return
  let active=true
  const read=()=>{void getDemoPipeline().then(engine=>{if(active)setArticles(engine.getPublishedArticles().sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||'')))})}
  read();window.addEventListener('creata:pipeline-demo:change',read);window.addEventListener('storage',read)
  return()=>{active=false;window.removeEventListener('creata:pipeline-demo:change',read);window.removeEventListener('storage',read)}
 },[initial])
 return demoMode?articles:initial
}
export function useDemoPipelineState(){
 const[state,setState]=useState<DemoState>()
 useEffect(()=>{
  if(!demoMode)return
  let active=true
  const read=()=>{void getDemoPipeline().then(engine=>{if(active)setState(engine.getSnapshot())})}
  read();window.addEventListener('creata:pipeline-demo:change',read);window.addEventListener('storage',read)
  return()=>{active=false;window.removeEventListener('creata:pipeline-demo:change',read);window.removeEventListener('storage',read)}
 },[])
 return state
}
