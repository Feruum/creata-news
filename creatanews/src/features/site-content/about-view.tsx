import Link from 'next/link'
import {ArrowUpRight} from 'lucide-react'
import type {AboutContent,Locale} from '@creata/contracts'
import {copy,localePath} from '@/lib/i18n'
import {Newsletter} from '@/components/cards'
export function AboutView({locale,about,preview=false}:{locale:Locale;about:AboutContent;preview?:boolean}){
 const content=about[locale]
 const t=copy[locale]
 return <>
  {preview&&<div className='preview-banner'><strong>{t.preview}</strong>{t.previewNote}</div>}
  <div className='page-heading'><p className='eyebrow'>{t.about}</p><h1>{content.title}</h1></div>
  <article className='about-content'>
   <p>{content.body}</p>
   <p>{content.second}</p>
   <Link className='button button-outline' href={localePath(locale,'/ecosystem')}>{t.ecosystem}<ArrowUpRight size={16} aria-hidden='true'/></Link>
   <div className='about-values'>{content.values.map(([title,text],i)=><section key={title}><span className='eyebrow'>0{i+1}</span><h2>{title}</h2><p>{text}</p></section>)}</div>
  </article>
  <Newsletter locale={locale}/>
 </>
}
