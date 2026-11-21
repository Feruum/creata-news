import type { Metadata } from 'next'
import { cookies } from 'next/headers'
import { notFound } from 'next/navigation'
import type { NewspaperPublication } from '@creata/contracts'
import { apiOrigin } from '@/lib/api'
import { isLocale } from '@/lib/i18n'
import { StateMessage } from '@/components/state-message'
import { PublishedNewspaper } from '@/features/newspaper/published-issue'
import {demoMode} from '@/demo/config'
import {DemoNewspaperReader} from '@/demo/newspaper-reader'

export const metadata: Metadata = { title: 'Newspaper preview', robots: { index: false, follow: false } }
export default async function NewspaperPreview({ params }: { params: Promise<{ locale: string; id: string }> }) {
  if(demoMode){const{locale,id}=await params;return <DemoNewspaperReader locale={isLocale(locale)?locale:'ru'} previewId={id}/>}
  const { locale, id } = await params
  if (!isLocale(locale)) notFound()
  let response: Response
  try { response = await fetch(`${apiOrigin()}/api/creata/v1/admin/newspapers/${encodeURIComponent(id)}/preview`, { headers: { cookie: (await cookies()).toString() }, cache: 'no-store', signal: AbortSignal.timeout(10000) }) }
  catch { return <StateMessage locale={locale} error/> }
  if ([401,403,404].includes(response.status)) notFound()
  if (!response.ok) return <StateMessage locale={locale} error/>
  return <PublishedNewspaper issue={await response.json() as NewspaperPublication} locale={locale} preview/>
}
