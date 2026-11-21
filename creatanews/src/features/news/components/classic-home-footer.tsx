import Link from 'next/link'
import {ArrowUpRight} from 'lucide-react'
import {CreataBrand as Brand} from '@creata/brand/logo'
import type {Locale} from '@/lib/models'
import {copy,localePath} from '@/lib/i18n'
import {newspaperCopy} from '@/features/newspaper/catalog'

export function ClassicHomeFooter({locale}:{locale:Locale}){
 const t=copy[locale]
 return <footer className='classic-home-footer'><div className='classic-home-footer-grid'>
  <div className='classic-home-footer-identity'><Link className='classic-home-footer-brand' href={localePath(locale)} aria-label='Creata News'><Brand/></Link><div className='classic-home-footer-subscribe'><p>{t.newsForAction}</p><Link href={localePath(locale,'/profile#digest')}>{t.subscribe}<ArrowUpRight size={17}/></Link></div></div>
  <nav aria-label={t.articles}><strong>{t.articles}</strong><Link href={localePath(locale,'/latest')}>{t.allStories}</Link><Link href={localePath(locale,'/gazeta')}>{newspaperCopy[locale].title}</Link></nav>
  <nav aria-label={t.action}><strong>{t.action}</strong><Link href={localePath(locale,'/opportunities')}>{t.opportunities}</Link><Link href={localePath(locale,'/saved')}>{t.saved}</Link></nav>
  <nav aria-label={t.learnMore}><strong>Creata News</strong><Link href={localePath(locale,'/about')}>{t.about}</Link><Link href={localePath(locale,'/authors')}>{t.authors}</Link></nav>
 </div><div className='classic-home-footer-bottom'><span>© 2026 Creata News</span><span>{t.publicNote}</span></div></footer>
}
