'use client'
import useEmblaCarousel from 'embla-carousel-react'
import { useInView,useReducedMotion } from 'motion/react'
import { useCallback,useEffect,useId,useRef,useState,type KeyboardEvent } from 'react'
import type { Locale,Opportunity } from '@/lib/models'
import { CarouselActions } from '@/shared/components/carousel-actions'
import { bindCarouselWheel } from '@/shared/lib/carousel-wheel'
import { OpportunityCard } from './opportunity-card'
const labels={ru:{carousel:'Карусель возможностей',card:'Карточка'},kk:{carousel:'Мүмкіндіктер каруселі',card:'Карточка'},en:{carousel:'Opportunity carousel',card:'Slide'}}
export function OpportunityCarousel({items,locale,title}:{items:Opportunity[];locale:Locale;title:string}){
  const reduced=useReducedMotion(),root=useRef<HTMLDivElement>(null),inView=useInView(root,{amount:.1})
  const[viewport,api]=useEmblaCarousel({loop:true,align:'start',slidesToScroll:1,duration:reduced?0:30})
  const[selected,setSelected]=useState(0),[hasMore,setHasMore]=useState(false),[hovered,setHovered]=useState(false),[focused,setFocused]=useState(false),[dragging,setDragging]=useState(false),[visible,setVisible]=useState(true)
  const id=useId(),t=labels[locale]
  useEffect(()=>{
    const update=()=>setVisible(document.visibilityState==='visible');update();document.addEventListener('visibilitychange',update)
    return()=>document.removeEventListener('visibilitychange',update)
  },[])
  useEffect(()=>{
    if(!api)return
    const update=()=>{setSelected(api.selectedScrollSnap());setHasMore(api.scrollSnapList().length>1)},down=()=>setDragging(true),up=()=>setDragging(false)
    update();api.on('select',update).on('reInit',update).on('pointerDown',down).on('pointerUp',up)
    return()=>{api.off('select',update).off('reInit',update).off('pointerDown',down).off('pointerUp',up)}
  },[api])
  const scroll=useCallback((direction:-1|1)=>{
    if(!api)return
    if(direction===1){if(api.canScrollNext())api.scrollNext(!!reduced);else api.scrollTo(0,!!reduced)}
    else{if(api.canScrollPrev())api.scrollPrev(!!reduced);else api.scrollTo(api.scrollSnapList().length-1,!!reduced)}
  },[api,reduced])
  useEffect(()=>{
    if(!api||!hasMore)return
    return bindCarouselWheel(api.rootNode(),scroll)
  },[api,hasMore,scroll])
  const playing=!!api&&hasMore&&!reduced&&!hovered&&!focused&&!dragging&&visible&&inView
  useEffect(()=>{if(!playing)return;const timer=setInterval(()=>scroll(1),6000);return()=>clearInterval(timer)},[playing,scroll])
  function keyboard(event:KeyboardEvent){if(event.key==='ArrowRight'||event.key==='ArrowLeft'){event.preventDefault();scroll(event.key==='ArrowRight'?1:-1)}}
  return <div ref={root} className='opportunity-carousel' data-testid='opportunity-carousel' data-current-index={selected} data-autoplay={playing?'playing':'paused'} onMouseEnter={()=>setHovered(true)} onMouseLeave={()=>setHovered(false)} onFocusCapture={()=>setFocused(true)} onBlurCapture={event=>{if(!event.currentTarget.contains(event.relatedTarget as Node|null))setFocused(false)}}>
    {hasMore&&<div className='opportunity-carousel-actions'><CarouselActions title={title} trackId={id} canScroll={{previous:true,next:true}} scroll={scroll} locale={locale} children={null}/></div>}
    <div ref={viewport} id={id} className='opportunity-carousel-viewport' data-carousel-viewport role='region' aria-roledescription={t.carousel} aria-label={title} tabIndex={0} onKeyDown={keyboard}>
      <div className='opportunity-carousel-track'>
        {items.map((item,index)=><div className='opportunity-carousel-slide' key={item.id} role='group' aria-roledescription={t.card} aria-label={`${index+1} / ${items.length}`}><OpportunityCard opportunity={item} locale={locale}/></div>)}
      </div>
    </div>
  </div>
}
