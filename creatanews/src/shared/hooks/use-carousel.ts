import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from 'react'
import useEmblaCarousel from 'embla-carousel-react'
import { useReducedMotion } from 'motion/react'
import { bindCarouselWheel } from '@/shared/lib/carousel-wheel'

export function useCarousel<T extends HTMLElement = HTMLDivElement>(count: number,options?:Parameters<typeof useEmblaCarousel>[0]) {
  const track = useRef<T>(null)
  const reduced=useReducedMotion()
  const [viewport,api]=useEmblaCarousel({loop:true,align:'start',slidesToScroll:1,duration:reduced?0:30,...options})
  const [selected,setSelected]=useState(0),[snapCount,setSnapCount]=useState(0)
  const hasMore=snapCount>1
  const canScroll={previous:hasMore,next:hasMore}

  const scroll=useCallback((direction:-1|1)=>{
    if(!api)return
    // Embla may fall back to finite positioning when too few cards fill the
    // viewport. Navigation still wraps, as it does in opportunities.
    if(direction===1){if(api.canScrollNext())api.scrollNext(!!reduced);else api.scrollTo(0,!!reduced)}
    else{if(api.canScrollPrev())api.scrollPrev(!!reduced);else api.scrollTo(api.scrollSnapList().length-1,!!reduced)}
  },[api,reduced])

  useEffect(() => {
    if(!api)return
    const update=()=>{setSelected(api.selectedScrollSnap());setSnapCount(api.scrollSnapList().length)}
    update();api.on('select',update).on('reInit',update)
    return()=>{api.off('select',update).off('reInit',update)}
  },[api,count])

  useEffect(()=>{if(api&&hasMore)return bindCarouselWheel(api.rootNode(),scroll)},[api,hasMore,scroll])

  function keyboard(event:KeyboardEvent<HTMLElement>){
    if(event.target!==event.currentTarget)return
    if(event.key==='ArrowRight'||event.key==='ArrowLeft'){
      event.preventDefault();scroll(event.key==='ArrowRight'?1:-1)
    }
  }

  return { track, viewport, canScroll, scroll, keyboard, selected, snapCount }
}
