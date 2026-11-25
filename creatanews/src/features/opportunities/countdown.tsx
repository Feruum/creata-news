'use client'
// Original Countdown from creata-news-v2/source-v2/src/ui.jsx and FlowDigit
// from skiper37.jsx. Explicit locale and a mounted clock make SSR stable.
import NumberFlow from '@number-flow/react'
import { useReducedMotion } from 'motion/react'
import { useEffect,useState } from 'react'
import type { Locale } from '@/lib/models'
import styles from './opportunity-theme.module.css'
const units={ru:['д','ч','мин','с'],kk:['к','сағ','мин','с'],en:['d','h','m','s']}
export function countdownParts(iso:string,now:number){
  const deadline=new Date(iso).valueOf(),ms=Number.isFinite(deadline)?Math.max(0,deadline-now):0
  return [Math.floor(ms/864e5),Math.floor(ms/36e5)%24,Math.floor(ms/6e4)%60,Math.floor(ms/1e3)%60]
}
function FlowDigit({value,className}:{value:number|null;className:string}){
  const reduced=useReducedMotion()
  return value===null?<span className={className}>--</span>:<NumberFlow value={value} format={{minimumIntegerDigits:2}} className={className} animated={!reduced} locales='en-US'/>
}
export function Countdown({iso,locale}:{iso:string;locale:Locale}){
  const[now,setNow]=useState<number|null>(null)
  useEffect(()=>{setNow(Date.now());const timer=setInterval(()=>setNow(Date.now()),1000);return()=>clearInterval(timer)},[])
  const parts=now===null?[null,null,null,null]:countdownParts(iso,now)
  return <div className='flex gap-1.5 font-mono tabular-nums' role='timer' aria-live='off'>
    {parts.map((value,index)=><span key={index} className={`${styles.timerCell} flex min-w-[44px] flex-col items-center rounded-md px-1.5 py-1`}>
      <FlowDigit value={value} className='text-[18px] font-semibold leading-none'/>
      <span className='mt-0.5 text-[12px] opacity-60'>{units[locale][index]}</span>
    </span>)}
  </div>
}
