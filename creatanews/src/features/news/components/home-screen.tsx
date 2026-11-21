'use client'

import {ArrowUpRight} from 'lucide-react'
import Image from 'next/image'
import Link from 'next/link'
import {useEffect,useId,useRef,useState} from 'react'
import {useReducedMotion} from 'motion/react'
import {homeEaseOutCss} from '../use-home-motion'
import {useDemoArticles} from '@/demo/use-pipeline'
import type {Article,Category,Locale} from '@/lib/models'
import {copy,localePath} from '@/lib/i18n'
import {articleTopics} from '@/lib/topics'
import {CarouselActions} from '@/shared/components/carousel-actions'
import {useCarousel} from '@/shared/hooks/use-carousel'
import {StateMessage} from '@/components/state-message'
import {ArticleCard} from '@/components/cards'
import type {NewspaperIssue} from '@/features/newspaper/catalog'
import {homeLabels} from '../home-copy'
import {HomeFeatures} from './home-features'
import {homeStoryImage} from '../home-story-images'
import styles from './home-screen.module.css'

const carouselOptions={breakpoints:{'(min-width: 700px)':{active:false}}}
export function HomeScreen({articles:initialArticles,categories,locale,issues=[],error=false}:{articles:Article[];categories:Category[];locale:Locale;issues?:NewspaperIssue[];error?:boolean}){
 const articles=useDemoArticles(initialArticles),l=homeLabels[locale],t=copy[locale]
 const [topic,setTopic]=useState('all'),railId=useId(),reduced=useReducedMotion()
 const [filtersUsed,setFiltersUsed]=useState(false)
 const filterAnimation=useRef<Animation|null>(null),animateFilter=useRef(false),filterOpacity=useRef(.65)
 useEffect(()=>{
  const viewport=document.getElementById(railId)
  const requested=animateFilter.current
  animateFilter.current=false
  if(!viewport||!requested||reduced)return
  const animation=viewport.animate([{opacity:filterOpacity.current},{opacity:1}],{duration:180,easing:homeEaseOutCss})
  filterAnimation.current=animation
  animation.onfinish=()=>{if(filterAnimation.current===animation)filterAnimation.current=null}
  return()=>{animation.cancel();if(filterAnimation.current===animation)filterAnimation.current=null}
 },[topic,reduced,railId])
 function chooseTopic(id:string,pointer:boolean){
  if(id===topic)return
  const viewport=document.getElementById(railId)
  filterOpacity.current=filterAnimation.current&&viewport?Number(getComputedStyle(viewport).opacity):.65
  filterAnimation.current?.cancel()
  animateFilter.current=pointer&&!reduced
  setFiltersUsed(true)
  setTopic(id)
 }
 const filtered=topic==='all'?articles:articles.filter(article=>articleTopics(article,categories).some(item=>item.id===topic))
 const visible=error?[]:filtered.slice(0,6),carousel=useCarousel(visible.length,carouselOptions)
 const filters=[['all',l.all],['technology',l.technology],['education',l.science],['creative',l.creative],['impact',l.people]]
 return <div className={styles.home} data-testid='editorial-home-screen'>
  <section className={styles.hero} aria-labelledby='home-spotlight-title' data-testid='home-lead'>
   <Link href={localePath(locale,'/about')} className={`${styles.heroArtwork} home-intro-art`} tabIndex={-1} aria-hidden='true'><Image src='/images/home/creata-human-hero-v1.png' alt='' fill sizes='(min-width: 1440px) 870px, (min-width: 1100px) 66vw, (min-width: 700px) 57vw, 100vw' loading='eager'/></Link>
   <div className={styles.heroCopy}><div className='home-intro-title'><h1 id='home-spotlight-title'><Link href={localePath(locale,'/about')}>{l.hero}</Link></h1><p>{l.intro}</p></div><Link href={localePath(locale,'/about')} className={`${styles.heroByline} home-intro-byline`}><span aria-hidden='true'>≡</span><span>{t.authorUnknown}<small>Creata News</small></span></Link></div>
  </section>
  <section className={styles.latest} aria-labelledby='home-latest-heading'>
   <div className={styles.sectionHead}><h2 id='home-latest-heading'>{l.latest}</h2><div className={styles.sectionTools}><div className={styles.filters} role='group' aria-label={t.topics}>{filters.map(([id,label])=><button type='button' key={id} aria-pressed={topic===id} onClick={event=>chooseTopic(id,event.detail>0)}>{label}</button>)}</div><Link className={styles.seeAll} aria-label={t.allStories} href={`${localePath(locale,'/latest')}${topic==='all'?'':`?section=${topic}`}`}>{t.all}<ArrowUpRight size={13} aria-hidden='true'/></Link></div></div>
   {error?<StateMessage locale={locale} error/>:!articles.length?<StateMessage locale={locale}/>:!visible.length?<p className={styles.empty} role='status'>{l.empty}</p>:<>
    <div id={railId} ref={carousel.viewport} className={styles.storyViewport} data-home-filtered={filtersUsed||undefined} tabIndex={0} onKeyDown={carousel.keyboard} role='group' aria-label={l.latest} aria-roledescription='carousel' data-current-index={carousel.selected} data-snap-count={carousel.snapCount}><div ref={carousel.track} className={`home-editorial-grid ${styles.storyGrid}`} data-testid='home-story-rail'>{visible.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories} variant='editorial' imageSrc={homeStoryImage(article)}/>)}</div></div>
    <div className={styles.mobileActions}><CarouselActions title={l.latest} trackId={railId} canScroll={carousel.canScroll} scroll={carousel.scroll} locale={locale}><span>{String(carousel.selected+1).padStart(2,'0')} / {String(visible.length).padStart(2,'0')}</span></CarouselActions></div>
   </>}
  </section>
  <HomeFeatures articles={error?[]:articles} categories={categories} locale={locale} issues={issues}/>
 </div>
}
