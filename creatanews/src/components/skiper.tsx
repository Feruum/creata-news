'use client'
/* Adapted from free Skiper UI components skiper58 TextRoll, skiper89 reading
   progress, skiper54 carousel and skiper37 animated values. Source adaptations
   preserved from creata-news-v2. Attribution: https://skiper-ui.com/ */
import { motion, useReducedMotion, useScroll, useMotionValueEvent, animate } from 'motion/react'
import { useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { ArrowLeft, ArrowRight, ArrowUp } from 'lucide-react'
import type { Locale } from '@/lib/models'
import { copy } from '@/lib/i18n'
import { useCarousel } from '@/shared/hooks/use-carousel'
import carouselStyles from '@/shared/components/story-carousel.module.css'
export function TextRoll({children}: {children: string}) {
  const reduced = useReducedMotion()
  return <motion.span className='text-roll' initial='initial' whileHover={reduced ? undefined : 'hovered'} whileFocus={reduced ? undefined : 'hovered'} aria-label={children}>
    <span aria-hidden='true'>{Array.from(children).map((letter,i) => <motion.span key={i} variants={{initial:{y:0},hovered:{y:'-105%'}}} transition={{duration:.24,delay:i*.012}}>{letter === ' ' ? '\u00a0' : letter}</motion.span>)}</span>
    <span aria-hidden='true' className='text-roll-copy'>{Array.from(children).map((letter,i) => <motion.span key={i} variants={{initial:{y:'105%'},hovered:{y:0}}} transition={{duration:.24,delay:i*.012}}>{letter === ' ' ? '\u00a0' : letter}</motion.span>)}</span>
  </motion.span>
}
export function ReadingProgress({locale}: {locale: Locale}) {
  const {scrollYProgress} = useScroll()
  const [percent,setPercent] = useState(0)
  useMotionValueEvent(scrollYProgress,'change',value => setPercent(Math.round(Math.max(0,Math.min(1,value))*100)))
  return <button className='reading-progress' aria-label={`${copy[locale].readingProgress}: ${percent}%. ${copy[locale].backHome}`} onClick={() => window.scrollTo({top:0,behavior:window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant':'smooth'})}>
    <svg viewBox='0 0 48 48' aria-hidden='true'><circle cx='24' cy='24' r='19'/><circle className='progress-ring' cx='24' cy='24' r='19' pathLength='100' strokeDasharray={`${percent} 100`}/></svg><ArrowUp size={17}/><span>{percent}%</span>
  </button>
}
export function AnimatedNumber({value}: {value: number}) {
  const ref = useRef<HTMLSpanElement>(null)
  const reduced = useReducedMotion()
  useEffect(() => { if (!ref.current) return; if (reduced) {ref.current.textContent=String(value);return} const controls=animate(0,value,{duration:.6,onUpdate:latest=>{if(ref.current)ref.current.textContent=String(Math.round(latest))}}); return ()=>controls.stop() }, [value,reduced])
  return <span ref={ref}>{value}</span>
}
export function StoryCarousel({children,locale,label}: {children: ReactNode[];locale:Locale;label?:string}) {
  const id=useId(),carousel=useCarousel(children.length)
  return <div className='story-carousel' role='region' aria-roledescription='carousel' aria-label={label||copy[locale].related} data-current-index={carousel.selected}>
    <div className='carousel-viewport' id={id} ref={carousel.viewport} tabIndex={0} onKeyDown={carousel.keyboard} data-current-index={carousel.selected} data-snap-count={carousel.snapCount}><div className='carousel-track'>{children.map((child,i)=><div className={`carousel-slide ${carouselStyles.slide}`} key={i} role='group' aria-roledescription='slide' aria-label={`${i+1} / ${children.length}`}>{child}</div>)}</div></div>
    {(children.length>1||carousel.canScroll.previous||carousel.canScroll.next)&&<div className='carousel-controls'><button type='button' onClick={()=>carousel.scroll(-1)} disabled={!carousel.canScroll.previous} aria-controls={id} aria-label={copy[locale].previous}><ArrowLeft size={19} aria-hidden='true'/></button><button type='button' onClick={()=>carousel.scroll(1)} disabled={!carousel.canScroll.next} aria-controls={id} aria-label={copy[locale].next}><ArrowRight size={19} aria-hidden='true'/></button></div>}
  </div>
}
