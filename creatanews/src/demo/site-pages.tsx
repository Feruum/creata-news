'use client'
import type {AboutContent,EcosystemContent,Locale} from '@creata/contracts'
import {StateMessage} from '@/components/state-message'
import {AboutView} from '@/features/site-content/about-view'
import {EcosystemView} from '@/features/site-content/ecosystem-view'
import {copy} from '@/lib/i18n'
import {useDemoPipelineState} from './use-pipeline'
export function DemoAbout({locale,preview=false}:{locale:Locale;preview?:boolean}){
 const state=useDemoPipelineState()
 if(!state)return <div className='state-message' role='status'>{copy[locale].loading}</div>
 const page=preview?state.pages.find(page=>page.slug==='about'):state.pagePublications.about
 return page?<AboutView locale={locale} preview={preview} about={page.content as AboutContent}/>:<StateMessage locale={locale}/>
}
export function DemoEcosystem({locale,preview=false}:{locale:Locale;preview?:boolean}){
 const state=useDemoPipelineState()
 if(!state)return <div className='state-message' role='status'>{copy[locale].loading}</div>
 const page=preview?state.pages.find(page=>page.slug==='ecosystem'):state.pagePublications.ecosystem
 return page?<EcosystemView locale={locale} preview={preview} content={page.content as EcosystemContent} articles={Object.values(state.publications).filter(article=>article.layer==='eco')} categories={state.categories}/>:<StateMessage locale={locale}/>
}
