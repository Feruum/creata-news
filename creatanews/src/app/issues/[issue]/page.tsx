import { cookies } from 'next/headers'
import { notFound, redirect } from 'next/navigation'
import { isLocale } from '@/lib/i18n'
import { getPublishedNewspaper } from '@/features/newspaper/public-api'
import { PublicApiError } from '@/lib/api'

export default async function IssueEntry({ params }: { params: Promise<{ issue: string }> }) {
  const { issue } = await params
  try { await getPublishedNewspaper(issue) } catch (error) { if (error instanceof PublicApiError && error.status === 404) notFound(); throw error }
  const saved = (await cookies()).get('creata-locale')?.value
  redirect(`/${saved && isLocale(saved) ? saved : 'ru'}/gazeta/${issue}`)
}
