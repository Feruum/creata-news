/** Add snap-by-card trackpad input to the existing native/Embla carousels. */
export function bindCarouselWheel(element:HTMLElement,scroll:(direction:-1|1)=>void){
  let lastEvent=0,direction=0,distance=0,handled=false
  function wheel(event:WheelEvent){
    // Keep vertical page scrolling and trackpad pinch zoom native.
    if(event.ctrlKey||event.metaKey||event.defaultPrevented)return
    const delta=event.shiftKey&&!event.deltaX?event.deltaY:event.deltaX
    if(!delta||(!event.shiftKey&&Math.abs(delta)<=Math.abs(event.deltaY)))return
    event.preventDefault()
    const now=performance.now(),nextDirection=delta>0?1:-1
    if(now-lastEvent>180||direction!==nextDirection){distance=0;handled=false}
    lastEvent=now;direction=nextDirection
    const unit=event.deltaMode===1?16:event.deltaMode===2?element.clientWidth:1
    distance+=Math.abs(delta)*unit
    // Treat the momentum tail as part of the same swipe, not another card.
    if(!handled&&distance>=24){handled=true;scroll(nextDirection)}
  }
  element.addEventListener('wheel',wheel,{passive:false})
  return()=>element.removeEventListener('wheel',wheel)
}
