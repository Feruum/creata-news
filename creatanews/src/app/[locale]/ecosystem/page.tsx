import {EcosystemView} from '@/features/site-content/ecosystem-view'
import {demoMode} from '@/demo/config'
import {DemoEcosystem} from '@/demo/site-pages'
import Link from 'next/link'
import type {Metadata} from 'next'
import {ArrowDown,ArrowUpRight} from 'lucide-react'
import {getArticleCards,getCategories} from '@/lib/api'
import {copy,isLocale,localePath} from '@/lib/i18n'
import {publicArticles} from '@/lib/content'
import {ArticleCard} from '@/components/cards'
import {getSitePage} from '@/features/site-content/api'
import type {EcosystemContent} from '@creata/contracts'
import {StateMessage} from '@/components/state-message'
import {EcosystemGallery,EcosystemNumbers,EcosystemProducts,EcosystemStack} from '@/features/ecosystem/visuals'
import styles from '@/features/ecosystem/ecosystem.module.css'

type Props={params:Promise<{locale:string}>;searchParams:Promise<{preview?:string}>}
export async function generateMetadata({params}:Props):Promise<Metadata>{const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';try{const content=(await getSitePage('ecosystem')).content as EcosystemContent,t=content.copy[locale];return {title:t.title,description:t.description,openGraph:{title:t.title,description:t.description}}}catch{return {robots:{index:false,follow:false}}}}
export default async function Page({params,searchParams}:Props){
 const{locale:raw}=await params,locale=isLocale(raw)?raw:'ru',preview=(await searchParams).preview==='1'
 if(demoMode)return <DemoEcosystem locale={locale} preview={preview}/>
 let content:EcosystemContent;try{content=(await getSitePage('ecosystem',preview)).content as EcosystemContent}catch{return <StateMessage locale={locale} error/>}
 const t=content.copy[locale]
 const[result,categories]=await Promise.all([getArticleCards(locale,{layer:'eco'}),getCategories()])
 const articles=publicArticles(result.data,locale)
 return <EcosystemView locale={locale} content={content} articles={articles} categories={categories.data} preview={preview}/>
}
