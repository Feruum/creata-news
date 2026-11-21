import {LaunchCatalog} from '@/features/launches/catalog-view'
import {DemoLaunches} from '@/demo/catalogs'
import {demoMode} from '@/demo/config'
import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { AnimatedNumber } from '@/components/skiper'
import { StateMessage } from '@/components/state-message'
import { getCategories } from '@/lib/api'
import { copy, isLocale, localePath, localize } from '@/lib/i18n'
import { regionLabel } from '@/lib/topics'
import { getLaunches } from '@/features/launches/data'
import { launchCopy,launchSocialCopy } from '@/features/launches/copy'
import { LaunchCard } from '@/features/launches/launch-card'
import { LaunchVotesProvider } from '@/features/launches/votes'
import styles from '@/features/launches/launches.module.css'

type Props={params:Promise<{locale:string}>;searchParams:Promise<{q?:string|string[];category?:string|string[];country?:string|string[];date?:string|string[];sort?:string|string[]}>}
export async function generateMetadata({params}:Props):Promise<Metadata> {
  const {locale}=await params
  if(!isLocale(locale))return {}
  return {title:launchCopy[locale].title,description:launchCopy[locale].intro,alternates:{canonical:localePath(locale,'/launches'),languages:{ru:'/ru/launches',kk:'/kk/launches',en:'/en/launches'}}}
}
export default async function Page({params,searchParams}:Props) {
  const {locale}=await params
  if(!isLocale(locale))notFound()
  const query=await searchParams,t=copy[locale],l=launchCopy[locale],s=launchSocialCopy[locale]
  const first=(value:string|string[]|undefined)=>Array.isArray(value)?value[0]:value
  const filters={q:first(query.q),category:first(query.category),country:first(query.country),date:first(query.date),sort:first(query.sort)||'popular'}
  if(demoMode)return <DemoLaunches locale={locale} filters={filters}/>
  const [all,categories]=await Promise.all([getLaunches(locale),getCategories()])
 return <LaunchCatalog locale={locale} data={all.data} categories={categories.data} filters={filters} error={all.error}/>
}
