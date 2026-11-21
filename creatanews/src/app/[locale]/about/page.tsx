import {isLocale} from '@/lib/i18n'
import {getSitePage} from '@/features/site-content/api'
import {AboutView} from '@/features/site-content/about-view'
import {StateMessage} from '@/components/state-message'
import type {AboutContent} from '@creata/contracts'
import {demoMode} from '@/demo/config'
import {DemoAbout} from '@/demo/site-pages'
export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{preview?:string}>}){
 const{locale:raw}=await params,locale=isLocale(raw)?raw:'ru',preview=(await searchParams).preview==='1'
 if(demoMode)return <DemoAbout locale={locale} preview={preview}/>
 try{return <AboutView locale={locale} preview={preview} about={(await getSitePage('about',preview)).content as AboutContent}/>}catch{return <StateMessage locale={locale} error/>}
}
