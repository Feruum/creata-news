'use client'

import {useEffect} from 'react'
import {animate,inView} from 'motion/react'

export const homeEaseOut=[.23,1,.32,1] as const
export const homeEaseOutCss=`cubic-bezier(${homeEaseOut.join(',')})`

export function useHomeMotion(enabled:boolean){
 useEffect(()=>{
  if(!enabled)return
  const roots=Array.from(document.querySelectorAll<HTMLElement>('.home-shell'))
  const preference=window.matchMedia('(prefers-reduced-motion: reduce)')
  const seen=new WeakSet<HTMLElement>()
  const observers=new Map<HTMLElement,()=>void>()
  const animations=new Set<{complete:()=>void}>()
  let keyboard=false
  const finish=()=>{for(const animation of animations)animation.complete();animations.clear()}
  const register=()=>{
   for(const [node,stop] of observers)if(!node.isConnected){stop();observers.delete(node)}
   for(const root of roots)for(const node of root.querySelectorAll<HTMLElement>('[data-home-reveal]')){
    if(seen.has(node)||observers.has(node))continue
    if(preference.matches){seen.add(node);continue}
    observers.set(node,inView(node,()=>{
     seen.add(node)
     if(preference.matches||keyboard||node.contains(document.activeElement)||(node.dataset.homeReveal==='card'&&node.closest('[data-home-filtered]')))return
     const card=node.dataset.homeReveal==='card'
     const targets=card?Array.from(node.querySelectorAll<HTMLElement>('.card-image-link')):[node]
     const order=card&&node.parentElement?Array.from(node.parentElement.children).indexOf(node)%3:0
     const frames=node.dataset.homeReveal==='fade'?{opacity:[0,1]}:{opacity:[0,1],transform:['translateY(12px)','translateY(0px)']}
     const animation=animate(targets,frames,{duration:card ? .35 : .45,ease:homeEaseOut,delay:index=>order*.05+index*.04})
     animations.add(animation)
     void animation.then(()=>animations.delete(animation))
    },{amount:.15,margin:'0px 0px 64px 0px'}))
   }
  }
  const onKeyboard=()=>{keyboard=true;document.documentElement.dataset.homeInput='keyboard';finish()}
  const onPointer=()=>{if(keyboard){keyboard=false;delete document.documentElement.dataset.homeInput}}
  const onPreference=()=>{
   if(preference.matches){finish();for(const stop of observers.values())stop();observers.clear()}
   register()
  }
  const changes=new MutationObserver(register)
  for(const root of roots)changes.observe(root,{childList:true,subtree:true})
  document.addEventListener('keydown',onKeyboard,true)
  document.addEventListener('pointerdown',onPointer,{capture:true,passive:true})
  document.addEventListener('pointermove',onPointer,{passive:true})
  preference.addEventListener('change',onPreference)
  register()
  return()=>{
   changes.disconnect();for(const stop of observers.values())stop();finish()
   document.removeEventListener('keydown',onKeyboard,true)
   document.removeEventListener('pointerdown',onPointer,true)
   document.removeEventListener('pointermove',onPointer)
   preference.removeEventListener('change',onPreference)
   delete document.documentElement.dataset.homeInput
  }
 },[enabled])
}
