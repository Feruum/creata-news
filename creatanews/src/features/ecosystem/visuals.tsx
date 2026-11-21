'use client'
import Link from 'next/link'
import {ArrowUp,ArrowUpRight} from 'lucide-react'
import {motion,useReducedMotion,useScroll,useTransform,type MotionValue} from 'motion/react'
import {useRef,useState} from 'react'
import {AnimatedNumber,TextRoll} from '@/components/skiper'
import {localePath,localize} from '@/lib/i18n'
import type {Locale} from '@/lib/models'
import type {EcosystemContent} from '@creata/contracts'
import styles from './ecosystem.module.css'

/* Reused: source-v2/src/skiper/skiper52.jsx HoverExpand_001 and skiper41.jsx
   ProgressiveBlur. @gurvinder-singh02, https://skiper-ui.com/. Buttons add
   keyboard/touch parity; flexible widths fit the existing responsive shell. */
export function EcosystemGallery({locale,content}:{locale:Locale;content:EcosystemContent}){
 const ecosystemCopy=content.copy,ecosystemPhotos=content.photos
 const[active,setActive]=useState(0),reduced=useReducedMotion()
 return <div className={styles.gallery} aria-label={ecosystemCopy[locale].stories}>
  {ecosystemPhotos.map((image,index)=><motion.button key={image.src} type='button' className={styles.galleryPhoto} initial={false} animate={{flexGrow:active===index?5:1}} transition={{duration:reduced?0:.4,ease:[.22,1,.36,1]}} onMouseEnter={()=>setActive(index)} onFocus={()=>setActive(index)} onClick={()=>setActive(index)} aria-pressed={active===index} aria-label={`${image.label}. ${localize(image.caption,locale)}`}>
   <img src={image.src} alt='' width={800} height={600} fetchPriority={index===0?'high':'auto'}/><span className={styles.progressiveBlur} aria-hidden='true'/><span className={styles.photoNumber}>0{index+1}</span>
   <span className={styles.photoLabel}><strong>{image.label}</strong>{active===index&&<motion.span initial={false} animate={{opacity:1}}>{localize(image.caption,locale)}</motion.span>}</span>
  </motion.button>)}
 </div>
}
export function EcosystemNumbers({locale,content}:{locale:Locale;content:EcosystemContent}){
 const ecosystemCopy=content.copy,ecosystemLayers=content.layers
 const t=ecosystemCopy[locale],count=ecosystemLayers.reduce((sum,layer)=>sum+layer.companies.length,0)
 return <dl className={styles.numbers}>{[{value:count,label:t.companies},{value:ecosystemLayers.length,label:t.layers},{value:1,label:t.route}].map(item=><div key={item.label}><dt>{item.label}</dt><dd><AnimatedNumber value={item.value}/><span aria-hidden='true'>↗</span></dd></div>)}</dl>
}

/* Actual Skiper16 StickyCard_001 scroll/scale mechanic from the supplied
   prototype. Phones and reduced motion use normal document flow through CSS. */
function LayerCard({index,progress,locale,layer,total}:{index:number;progress:MotionValue<number>;locale:Locale;layer:EcosystemContent["layers"][number];total:number}){
 const scale=useTransform(progress,[index/total,1],[1,1-(total-index-1)*.035]),reduced=useReducedMotion()
 return <div className={styles.stackSlot} style={{top:`${84+index*16}px`}}><motion.article id={`ecosystem-${layer.id}`} className={styles.layerCard} style={{scale:reduced?1:scale}} aria-labelledby={`layer-title-${layer.id}`}>
  <figure className={styles.layerFigure}><img src={layer.photo} alt={localize(layer.caption,locale)} loading='lazy' width={1000} height={900}/><figcaption>{localize(layer.caption,locale)}</figcaption><span className={styles.layerFigureNumber} aria-hidden='true'>{layer.number}</span></figure>
  <div className={styles.layerContent}><div className={styles.layerEyebrow}><span>{layer.number} / {String(total).padStart(2,'0')}</span><span>{localize(layer.subtitle,locale)}</span></div><h3 id={`layer-title-${layer.id}`}>{localize(layer.title,locale)}</h3><p>{localize(layer.description,locale)}</p>
   <ul className={styles.companyList}>{layer.companies.map(company=><li key={company.name}><a href={company.href} target='_blank' rel='noopener noreferrer'><span><strong>{company.name}</strong><small>{localize(company.note,locale)}</small></span><ArrowUpRight size={19} aria-hidden='true'/></a></li>)}</ul>
  </div>
 </motion.article></div>
}
export function EcosystemStack({locale,content}:{locale:Locale;content:EcosystemContent}){
 const ecosystemLayers=content.layers
 const root=useRef<HTMLDivElement>(null),{scrollYProgress}=useScroll({target:root,offset:['start start','end end']})
 return <div ref={root} className={styles.stack}>{ecosystemLayers.map((layer,index)=><LayerCard key={layer.id} index={index} progress={scrollYProgress} locale={locale} layer={layer} total={ecosystemLayers.length}/>)}</div>
}

export function EcosystemProducts({locale,content}:{locale:Locale;content:EcosystemContent}){
 const ecosystemCopy=content.copy,ecosystemProducts=content.products
 const t=ecosystemCopy[locale],products=ecosystemProducts
 return <div className={styles.directory}>
  <div className={styles.productList}>{products.map(product=>{
   const internal=product.href.startsWith('/'),href=internal?localePath(locale,product.href):product.href
   const content=<><span className={styles.productIndex}>{String(ecosystemProducts.indexOf(product)+1).padStart(2,'0')}</span><span className={styles.productName}><small>{localize(product.audience,locale)}</small><strong><TextRoll>{product.name}</TextRoll></strong></span><span className={styles.productDescription}>{localize(product.description,locale)}</span><span className={styles.productArrow}><ArrowUpRight size={23} aria-hidden='true'/></span></>
   return internal?<Link className={styles.productRow} key={product.id} href={href} data-product-id={product.id}>{content}</Link>:<a className={styles.productRow} key={product.id} href={href} target='_blank' rel='noopener noreferrer' data-product-id={product.id}>{content}<span className='sr-only'> · {t.openProduct}</span></a>
  })}</div>
  <a className={styles.backToTop} href='#ecosystem-products'>{t.backToProducts}<ArrowUp size={15} aria-hidden='true'/></a>
 </div>
}
