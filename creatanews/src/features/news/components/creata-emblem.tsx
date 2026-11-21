'use client'

// Viewport entry and hover expansion adapted from free Skiper UI skiper52.
// https://skiper-ui.com/v1/skiper52 — original Creata ≡ geometry below.
import { motion,useInView } from 'motion/react'
import { useEffect,useRef,useState } from 'react'

export function CreataEmblem({className}:{className?:string}) {
  const element=useRef<HTMLDivElement>(null)
  const entered=useInView(element,{once:true,amount:.35})
  const [reduced,setReduced]=useState(false)
  const [canHover,setCanHover]=useState(false)
  const [hovered,setHovered]=useState(false)

  useEffect(()=>{
    const pointer=window.matchMedia('(hover: hover) and (pointer: fine)')
    const preference=window.matchMedia('(prefers-reduced-motion: reduce)')
    const update=()=>{setCanHover(pointer.matches);setReduced(preference.matches);if(!pointer.matches||preference.matches)setHovered(false)}
    update();pointer.addEventListener('change',update);preference.addEventListener('change',update)
    return()=>{pointer.removeEventListener('change',update);preference.removeEventListener('change',update)}
  },[])

  const active=hovered&&canHover&&!reduced
  const spring={type:'spring' as const,duration:.45,bounce:.15}
  return <motion.div ref={element} className={className} aria-hidden='true' data-creata-emblem data-state={reduced?'reduced':active?'hovered':entered?'visible':'waiting'}
    initial={false}
    animate={{transform:active?'perspective(800px) rotateX(5deg) rotateY(-8deg) scale(1.035)':'perspective(800px) rotateX(0deg) rotateY(0deg) scale(1)'}}
    transition={reduced?{duration:0}:spring}
    onHoverStart={()=>{if(canHover&&!reduced)setHovered(true)}} onHoverEnd={()=>setHovered(false)}>
    <motion.svg viewBox='0 0 280 265' focusable='false' initial='hidden' animate={entered||reduced?'visible':'hidden'} variants={{hidden:{},visible:{transition:{staggerChildren:reduced?0:.06}}}}>
      <motion.path d='M27 213 216 179 263 207 73 244Z' fill='#211b91' initial={false} animate={{opacity:active ? .32 : .24,transform:active?'translateY(5px)':'translateY(0px)'}} transition={reduced?{duration:0}:spring}/>
      <g transform='translate(21 18) rotate(-8 110 100)' stroke='#4b44f0' strokeWidth='1.2' strokeLinejoin='round'>
        {[0,65,130].map((y,index)=><g key={y} transform={`translate(0 ${y})`}>
          <motion.g variants={{hidden:{opacity:reduced?1:0,transform:reduced?'translateY(0px)':'translateY(22px)'},visible:{opacity:1,transform:'translateY(0px)',transition:{duration:reduced?0:.32,ease:[.23,1,.32,1]}}}}>
            <motion.g data-creata-plate={index} initial={false} animate={{transform:active?`translate(${(index-1)*4}px, ${(index-1)*12}px)`:'translate(0px, 0px)'}} transition={reduced?{duration:0}:spring}>
              <path d='M0 28 185 0 225 26 40 54Z' fill='#faf8f2'/>
              <path d='M40 54 225 26 225 47 40 75Z' fill='#ccc5ff'/>
              <path d='M0 28 40 54 40 75 0 49Z' fill='#a69af5'/>
              <path d='M4 29 185 2' fill='none' stroke='#fff' strokeWidth='1.5'/>
            </motion.g>
          </motion.g>
        </g>)}
      </g>
    </motion.svg>
  </motion.div>
}
