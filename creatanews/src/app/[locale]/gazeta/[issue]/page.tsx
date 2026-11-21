import {OriginalNewspaperView} from '@/features/newspaper/original-view'
import {demoMode} from '@/demo/config'
import {DemoNewspaperReader} from '@/demo/newspaper-reader'
import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, ArrowUpRight } from 'lucide-react'
import { isLocale, localePath } from '@/lib/i18n'
import { PublicApiError } from '@/lib/api'
import { StateMessage } from '@/components/state-message'
import { getPublishedNewspaper } from '@/features/newspaper/public-api'
import { PublishedNewspaper } from '@/features/newspaper/published-issue'
import { newspaperCopy, newspaperDate } from '@/features/newspaper/catalog'
import { IssuePhotography } from '@/features/newspaper/issue-photography'
import {getSitePage} from '@/features/site-content/api'
import type {OriginalIssueContent,NewspaperPublication} from '@creata/contracts'
import '@/features/newspaper/newspaper.css'
import '@/features/newspaper/issue-01.css'

type Params = { locale: string; issue: string }
export const dynamic = 'force-dynamic'
export async function generateMetadata({ params }: { params: Promise<Params> }): Promise<Metadata> {
  const { locale, issue: id } = await params
  if (!isLocale(locale)) return {}
  let issue;try { const publication = await getPublishedNewspaper(id); issue = { id: publication.slug, number: publication.number, date: publication.date, pages: publication.pages, language: publication.locale, cover: publication.cover.url, headline: publication.description || publication.title } } catch { return { robots: { index: false, follow: false } } }
  return {
    title: `Creata News №${issue.number} — ${newspaperCopy[locale].title}`,
    description: issue.headline,
    alternates: { canonical: localePath(locale, `/gazeta/${issue.id}`), languages: { ru: `/ru/gazeta/${issue.id}`, kk: `/kk/gazeta/${issue.id}`, en: `/en/gazeta/${issue.id}` } },
    openGraph: { type: 'article', title: `Creata News №${issue.number}`, description: issue.headline, images: [{ url: issue.cover }] },
  }
}
export default async function NewspaperIssue({ params,searchParams }: { params: Promise<Params>;searchParams:Promise<{preview?:string}> }) {
  const { locale, issue: id } = await params
  if (!isLocale(locale)) notFound()
  if(demoMode)return <DemoNewspaperReader locale={locale} slug={id} preview={(await searchParams).preview==='1'}/>
  let issue:NewspaperPublication;try {if(id==='01'&&(await searchParams).preview==='1'){const page=await getSitePage('newspaper-01',true);issue={...page.content as OriginalIssueContent,id:'original-01',format:'original',revision:page.revision,publishedRevision:page.revision,entries:[]}}else{issue=await getPublishedNewspaper(id)}}catch(error){if(error instanceof PublicApiError&&error.status===404)notFound();return <StateMessage locale={locale} error/>}
  if(issue.format!=='original')return <PublishedNewspaper issue={issue} locale={locale}/>
  return <OriginalNewspaperView issue={issue} locale={locale}/>
}
