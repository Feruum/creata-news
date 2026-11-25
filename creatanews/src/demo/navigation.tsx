'use client'
import {useState} from 'react'
import {X} from 'lucide-react'
import type {Locale} from '@/lib/models'
import {copy} from '@/lib/i18n'
import {demoCopy} from './config'
const names={ru:{short:'Демо',editorial:'Редакция',hermes:'Пульт Hermes'},kk:{short:'Демо',editorial:'Редакция',hermes:'Hermes басқаруы'},en:{short:'Demo',editorial:'Newsroom',hermes:'Hermes control'}}
export function DemoNavigation({locale}:{locale:Locale}){
 const t=names[locale],[dismissed,setDismissed]=useState(false)
 if(dismissed)return null
 return <div className='preview-banner demo-navigation' data-testid='demo-notice'><span className='demo-notice-copy'>{demoCopy[locale].notice}</span><span className='demo-notice-short' aria-label={demoCopy[locale].notice} title={demoCopy[locale].notice}>{t.short}</span><span className='demo-notice-links'><a href='/editorial/today' aria-label={t.editorial}>{t.editorial} <span aria-hidden='true'>↗</span></a><a href='/hermes/sources' aria-label={t.hermes}><span className='demo-hermes-long'>{t.hermes}</span><span className='demo-hermes-short'>Hermes</span> <span aria-hidden='true'>↗</span></a></span><button type='button' aria-label={copy[locale].close} onClick={()=>setDismissed(true)}><X size={18} aria-hidden='true'/></button></div>
}
