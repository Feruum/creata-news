'use client'
// Restored from the original shared/components/shell.tsx: compact site-header,
// three-item PrimaryNavigation, original language control and desktop/bottom
// responsive navigation. News/account data now use explicit locale + API props.
import { ArrowUpRight,Compass,House,Menu,Search,UserRound,X } from 'lucide-react'
import Link from 'next/link'
import { usePathname,useRouter } from 'next/navigation'
import { useEffect,useRef,useState,type ReactNode } from 'react'
import { animate,useReducedMotion } from 'motion/react'
import type { Article,Category,Locale } from '@/lib/models'
import { PopularStories } from '@/features/news/components/popular-stories'
import { copy,localePath,switchLocale } from '@/lib/i18n'
import { topics,topicLabel,regionLabel } from '@/lib/topics'
import { AuthProvider } from './auth-context'
import { CreataBrand } from '@creata/brand/logo'
import { TextRoll } from './skiper'
import { ThemeToggle } from '@/shared/components/theme-toggle'
import { newspaperCopy } from '@/features/newspaper/catalog'
import {graphicCopy} from '@creata/graphics'
import { launchCopy } from '@/features/launches/copy'
import {demoMode} from '@/demo/config'
import {DemoNavigation} from '@/demo/navigation'
import {homeLabels} from '@/features/news/home-copy'
import {useHomeMotion} from '@/features/news/use-home-motion'
import {homeStoryImage} from '@/features/news/home-story-images'
import {ClassicHomeFooter} from '@/features/news/components/classic-home-footer'

export function Brand({small=false}:{small?:boolean}){return <CreataBrand className={small?'brand-mark-small':''}/>} 
function PrimaryNavigation({className,locale}:{className:string;locale:Locale}){
  const t=copy[locale],path=usePathname()
  return <nav className={className} aria-label={t.sections}>{[{href:'',title:t.home,icon:House},{href:'/latest',title:t.latest,icon:Compass},{href:'/profile',title:t.profile,icon:UserRound}].map(({href,title,icon:Icon})=>{const destination=localePath(locale,href);const active=href?path.startsWith(destination):path===destination;return <Link href={destination} key={href} aria-label={title} aria-current={active?'page':undefined}><Icon aria-hidden='true' size={20}/><span><TextRoll>{title}</TextRoll></span></Link>})}</nav>
}
function SiteHeader({locale,regions,categories,popularArticles}:{locale:Locale;categories:Category[];regions:string[];popularArticles:Article[]}){
  const t=copy[locale],path=usePathname(),router=useRouter(),dialog=useRef<HTMLDialogElement>(null),[open,setOpen]=useState(false),reduced=useReducedMotion(),bodyOverflow=useRef<string|undefined>(undefined)
  const home=path===localePath(locale,'/v2'),classic=path===localePath(locale),l=homeLabels[locale]
  // Skiper 38's staggered mega-menu direction, composed with the existing native
  // dialog and free Skiper 58 TextRoll. Native focus containment remains intact.
  useEffect(()=>{
    const node=dialog.current
    if(!node)return
    let cancelled=false
    const controls:{stop:()=>void}[]=[]
    const unlock=()=>{if(bodyOverflow.current!==undefined){document.body.style.overflow=bodyOverflow.current;bodyOverflow.current=undefined}}
    const items=Array.from(node.querySelectorAll<HTMLElement>('.menu-explore-grid h2,.menu-explore-grid h3,.menu-explore-grid nav>a,.menu-bottom-row'))
    if(open){
      if(bodyOverflow.current===undefined)bodyOverflow.current=document.body.style.overflow
      document.body.style.overflow='hidden'
      if(!node.open){
        node.style.opacity=reduced?'1':'0'
        node.style.transform=reduced?'none':'translateY(-24px)'
        for(const item of items){item.style.opacity=reduced?'1':'0';item.style.transform=reduced?'none':'translateY(10px)'}
        node.showModal()
        node.scrollTop=0
      }
      node.dataset.state='opening'
      const panel=animate(node,{opacity:1,transform:'translateY(0px)'},{duration:reduced?0:.28,ease:[.23,1,.32,1]})
      controls.push(panel)
      items.forEach((item,index)=>controls.push(animate(item,{opacity:1,transform:'translateY(0px)'},{duration:reduced?0:.2,delay:reduced?0:.025+Math.min(index,7)*.016,ease:[.23,1,.32,1]})))
      void panel.then(()=>{if(!cancelled)node.dataset.state='open'})
    }else if(node.open){
      node.dataset.state='closing'
      const exit=animate(node,{opacity:0,transform:'translateY(-18px)'},{duration:reduced?0:.18,ease:[.23,1,.32,1]})
      controls.push(exit)
      void exit.then(()=>{if(!cancelled){node.close();node.dataset.state='closed';unlock()}})
    }else unlock()
    return()=>{cancelled=true;controls.forEach(control=>control.stop())}
  },[open,reduced])
  useEffect(()=>()=>{if(bodyOverflow.current!==undefined)document.body.style.overflow=bodyOverflow.current},[])
  useEffect(()=>setOpen(false),[path])
  const links=[{path:'',title:t.home},{path:'/v2',title:l.version2},{path:'/gazeta',title:newspaperCopy[locale].title},{path:'/charts',title:graphicCopy[locale].title},{path:'/ecosystem',title:t.ecosystem},{path:'/launches',title:launchCopy[locale].title},{path:'/opportunities',title:t.opportunities},{path:'/profile#digest',title:t.digest}]
  const learnLinks=[{path:'/about',title:t.about},{path:'/authors',title:t.authors},{path:'/profile',title:t.profile},{path:'/saved',title:t.saved}]
  const languages=<fieldset className='language-switch locale-options' aria-label={t.language}>{(['ru','en','kk'] as const).map(l=><button type='button' key={l} aria-pressed={locale===l} onClick={()=>{setOpen(false);router.push(switchLocale(path,l))}}>{l==='kk'?'ҚАЗ':l.toUpperCase()}</button>)}</fieldset>
  return <><header className='site-header'>
    {classic?<><nav className='classic-home-utility' aria-label={t.quickLinks}><Link href={localePath(locale,'/about')}>{t.about}</Link><Link href={localePath(locale,'/latest')}>{t.latest}</Link><Link href={localePath(locale,'/profile#digest')}>{t.digest}</Link><Link href={localePath(locale,'/v2')}>{l.version2}<ArrowUpRight size={13} aria-hidden='true'/></Link></nav><div className='classic-home-masthead'><nav className='classic-home-masthead-topics' aria-label={t.topics}>{topics.slice(0,2).map(topic=><Link key={topic.id} href={localePath(locale,'/latest')+'?section='+topic.id}>{topicLabel(topic,locale)}</Link>)}</nav><Link href={localePath(locale)} aria-label='Creata News' className='brand brand-link'><Brand/></Link><nav className='classic-home-masthead-topics' aria-label={t.sections}><Link href={localePath(locale,'/ecosystem')}>{t.ecosystem}</Link><Link href={localePath(locale,'/opportunities')}>{t.opportunities}</Link></nav></div></>:<><Link href={localePath(locale,home?'/v2':'')} aria-label='Creata News' className='brand brand-link'><Brand/></Link>{home?<nav className='home-header-nav' aria-label={t.sections}><Link href={localePath(locale,'/latest')}>{l.materials}</Link><Link href={localePath(locale,'/opportunities')}>{l.opportunities}</Link><Link href={localePath(locale,'/authors')}>{l.peopleNav}</Link><Link href={localePath(locale,'/about')}>{l.about}</Link></nav>:<PrimaryNavigation className='desktop-nav' locale={locale}/>}</>}
    <div className='header-controls'>{classic&&<Link className='classic-home-profile-link' href={localePath(locale,'/profile')}>{t.profile}</Link>}{home&&<><Link className='home-search-link' href={localePath(locale,'/search')} aria-label={t.search}><Search size={19} aria-hidden='true'/></Link><Link className='home-paper-link' href={localePath(locale,'/gazeta')} aria-label={l.readPaper}><span className='home-paper-full'>{l.readPaper}</span><span className='home-paper-short'>{newspaperCopy[locale].title}</span><ArrowUpRight size={15} aria-hidden='true'/></Link></>}{!home&&<ThemeToggle locale={locale}/>} {!home&&languages}<button className='original-menu-trigger' aria-label={t.menu} aria-expanded={open} aria-controls='creata-menu' onClick={()=>setOpen(true)}><Menu size={21}/></button></div>
  </header>
  <dialog ref={dialog} id='creata-menu' className='original-menu-dialog' aria-labelledby='menu-heading' onCancel={event=>{event.preventDefault();setOpen(false)}} onClose={()=>setOpen(false)} onClick={event=>{if(event.target===event.currentTarget)setOpen(false)}}>
    <div className='original-menu-panel'>
      <header className='menu-top-row'>
        <nav aria-label={t.quickLinks}><Link href={localePath(locale,'/about')} onClick={()=>setOpen(false)}>{t.about}</Link><Link href={localePath(locale,'/latest')} onClick={()=>setOpen(false)}>{t.latest}</Link><Link href={localePath(locale,'/profile#digest')} onClick={()=>setOpen(false)}>{t.digest}</Link></nav>
        <form action={localePath(locale,'/search')} onSubmit={()=>setOpen(false)} className='search-box'><label htmlFor='menu-query' className='sr-only'>{t.search}</label><input id='menu-query' name='q' placeholder={t.search} autoFocus/><button type='submit' aria-label={t.search}><Search size={23} aria-hidden='true'/></button></form>
        <button className='menu-close' aria-label={t.close} onClick={()=>setOpen(false)}><X size={28} aria-hidden='true'/></button>
      </header>
      <div className='menu-explore-grid'>
        <h2 id='menu-heading'>{t.explore}</h2>
        <nav aria-label={t.topics}><h3>{t.topics}</h3>{topics.map(topic=><Link key={topic.id} href={`${localePath(locale,'/latest')}?section=${topic.id}`} onClick={()=>setOpen(false)}><TextRoll>{topicLabel(topic,locale)}</TextRoll></Link>)}</nav>
        <nav aria-label={t.regions}><h3>{t.regions}</h3><Link href={localePath(locale,'/latest')} onClick={()=>setOpen(false)}>{t.allRegions}</Link>{regions.map(region=><Link key={region} href={`${localePath(locale,'/latest')}?region=${encodeURIComponent(region)}`} onClick={()=>setOpen(false)}>{regionLabel(region,locale)}</Link>)}</nav>
        <nav aria-label={t.sections}><h3>{t.sections}</h3>{links.map(link=><Link key={link.path} href={localePath(locale,link.path)} onClick={()=>setOpen(false)}><TextRoll>{link.title}</TextRoll></Link>)}</nav>
      </div>
      <div className='menu-bottom-row'><span className='eyebrow'>{t.learnMore}</span><nav aria-label={t.learnMore}>{learnLinks.map(link=><Link key={link.path} href={localePath(locale,link.path)} onClick={()=>setOpen(false)}>{link.title}</Link>)}</nav></div>
      {home&&<details className='home-menu-popular'><summary>{l.popular}<ArrowUpRight size={16} aria-hidden='true'/></summary><PopularStories locale={locale} articles={popularArticles} categories={categories} imageForArticle={homeStoryImage}/></details>}
      <div className='menu-bottom-row menu-preferences'><Brand small/><p>{t.newsForAction}</p><div className='menu-preference-controls'>{languages}<ThemeToggle locale={locale}/></div></div>
    </div>
  </dialog></>
}
function HomeFooter({locale}:{locale:Locale}){
 const t=copy[locale],l=homeLabels[locale]
 return <footer className='home-footer'><div className='home-footer-inner'><Link className='home-footer-brand' href={localePath(locale,'/v2')} aria-label='Creata News'><Brand/></Link><div className='home-footer-grid'>
  <div className='home-footer-subscribe'><p>{l.footer}</p><Link href={localePath(locale,'/profile#digest')}><span>{t.subscribe}</span><ArrowUpRight size={18} aria-hidden='true'/></Link></div>
  <nav aria-label={l.learn}><strong>{l.learn}</strong><Link href={localePath(locale,'/latest')}>{t.allStories}</Link><Link href={localePath(locale,'/gazeta')}>{newspaperCopy[locale].title}</Link><Link href={localePath(locale,'/ecosystem')}>{t.ecosystem}</Link></nav>
  <nav aria-label={l.act}><strong>{l.act}</strong><Link href={localePath(locale,'/opportunities')}>{t.opportunities}</Link><Link href={localePath(locale,'/saved')}>{t.saved}</Link><Link href={localePath(locale,'/profile')}>{t.profile}</Link></nav>
  <nav aria-label={t.learnMore}><strong>Creata News</strong><Link href={localePath(locale,'/about')}>{t.about}</Link><Link href={localePath(locale,'/authors')}>{t.authors}</Link></nav>
 </div><div className='home-footer-bottom'><span>© 2026 Creata News</span><span>{t.publicNote}</span></div></div></footer>
}
export function Shell({locale,categories,regions,popularArticles,children}:{locale:Locale;categories:Category[];regions:string[];popularArticles:Article[];children:ReactNode}){
  const path=usePathname(),home=path===localePath(locale,'/v2'),classic=path===localePath(locale)
  useHomeMotion(home)
  useEffect(()=>{document.documentElement.lang=locale},[locale])
  return <AuthProvider><div className={`app-shell ${home?'home-shell':classic?'catalog-shell classic-home-shell':'catalog-shell'}`}><a href='#main-content' className='skip-link'>{copy[locale].skip}</a>{demoMode&&<DemoNavigation locale={locale}/> }<SiteHeader locale={locale} categories={categories} regions={regions} popularArticles={popularArticles}/><main id='main-content' tabIndex={-1}>{children}{!home&&<PopularStories locale={locale} articles={popularArticles} categories={categories} imageForArticle={classic?homeStoryImage:undefined}/>}</main>{!home&&!classic&&<footer className='original-site-footer'><Brand small/><span>{copy[locale].newsForAction}</span></footer>}{!home&&<PrimaryNavigation className='bottom-nav' locale={locale}/>}</div>{home&&<HomeFooter locale={locale}/>} {classic&&<ClassicHomeFooter locale={locale}/>}</AuthProvider>
}
