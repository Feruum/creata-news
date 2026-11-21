import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { isLocale, localePath } from '@/lib/i18n'
import { NewspaperArchive } from '@/features/newspaper/archive'
import { newspaperCopy } from '@/features/newspaper/catalog'
import { getNewspaperArchive } from '@/features/newspaper/public-api'
import {demoMode} from '@/demo/config'
import {DemoNewspaperArchive} from '@/demo/catalogs'

export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params
  if (!isLocale(locale)) return {}
  return { title: newspaperCopy[locale].title, description: newspaperCopy[locale].intro, alternates: { canonical: localePath(locale, '/gazeta') } }
}
export default async function Newspaper({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params
  if (!isLocale(locale)) notFound()
  if(demoMode)return <DemoNewspaperArchive locale={locale}/>
  const issues = await getNewspaperArchive()
  return <NewspaperArchive locale={locale} issues={issues.data} error={issues.error}/>
}
