import type {AboutContent,Locale} from '@creata/contracts'
import {copy} from '@/lib/i18n'
import {Newsletter} from '@/components/cards'
export function AboutView({locale,about,preview=false}:{locale:Locale;about:AboutContent;preview?:boolean}){
 const content=about[locale]
 return <>{preview&&<div className='preview-banner'><strong>{copy[locale].preview}</strong>{copy[locale].previewNote}</div>}<div className='page-heading'><p className='eyebrow'>{copy[locale].about}</p><h1>{content.title}</h1></div><article className='about-content'><p>{content.body}</p><p>{content.second}</p><div className='about-values'>{content.values.map(([title,text],i)=><section key={title}><span className='eyebrow'>0{i+1}</span><h2>{title}</h2><p>{text}</p></section>)}</div></article><Newsletter locale={locale}/></>
}
