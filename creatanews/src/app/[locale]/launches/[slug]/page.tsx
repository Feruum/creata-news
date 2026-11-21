import {LaunchDetail} from '@/features/launches/detail-view'
import {DemoLaunchReader} from '@/demo/catalogs'
import {demoMode} from '@/demo/config'
import type { Metadata } from 'next'
import Link from 'next/link'
import { notFound } from 'next/navigation'
import { ArrowLeft,ArrowUpRight } from 'lucide-react'
import { PublicApiError,getCategories } from '@/lib/api'
import { copy,formatDate,isLocale,localePath,localize } from '@/lib/i18n'
import { regionLabel } from '@/lib/topics'
import { StateMessage } from '@/components/state-message'
import { MediaImage } from '@/components/media-image'
import { StoryCarousel } from '@/components/skiper'
import { getLaunch,getLaunches } from '@/features/launches/data'
import { launchCopy } from '@/features/launches/copy'
import { LaunchCard } from '@/features/launches/launch-card'
import { LaunchVoteButton,LaunchVotesProvider } from '@/features/launches/votes'
import { LaunchDiscussion } from '@/features/launches/discussion'
import styles from '@/features/launches/launches.module.css'

type Props={params:Promise<{locale:string;slug:string}>}
export async function generateMetadata({params}:Props):Promise<Metadata> {
  const {locale,slug}=await params
  if(!isLocale(locale))return {}
  try{
    const launch=await getLaunch(slug)
    return {title:launch.name,description:localize(launch.tagline,locale),alternates:{canonical:localePath(locale,`/launches/${slug}`),languages:Object.fromEntries(['ru','kk','en'].map(lang=>[lang,`/${lang}/launches/${slug}`]))},openGraph:{title:launch.name,description:localize(launch.tagline,locale),...(launch.photo?{images:[launch.photo]}:{})}}
  }catch{return {title:launchCopy[locale].title,robots:{index:false,follow:false}}}
}
export default async function Page({params}:Props) {
  const {locale,slug}=await params
  if(!isLocale(locale))notFound()
  const l=launchCopy[locale],t=copy[locale]
  if(demoMode)return <DemoLaunchReader locale={locale} slug={slug}/>
  let launch
  try{launch=await getLaunch(slug)}catch(error){if(error instanceof PublicApiError&&error.status===404)notFound();return <StateMessage locale={locale} error/>}
  if(launch.status!=='published')notFound()
  const [others,categories]=await Promise.all([getLaunches(locale),getCategories()])
 return <LaunchDetail locale={locale} launch={launch} others={others.data} categories={categories.data}/>
}
