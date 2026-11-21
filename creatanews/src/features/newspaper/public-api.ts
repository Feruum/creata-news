import { cache } from 'react'
import type { NewspaperPublication } from '@creata/contracts'
import { loadList, serverApi } from '@/lib/api'
import type { NewspaperIssue } from './catalog'

export const getPublishedNewspaper = cache((slug: string) => serverApi<NewspaperPublication>(`/public/newspapers/${encodeURIComponent(slug)}`))
export async function getNewspaperArchive() {
  const result = await loadList<NewspaperPublication>('/public/newspapers')
  const editions: NewspaperIssue[] = result.data.map(issue => ({
    id: issue.slug, number: issue.number, date: issue.date, pages: issue.pages,
    language: issue.locale, cover: issue.cover.url, title: issue.title,
    headline: issue.description || issue.entries[0]?.article.title[issue.locale] || issue.title, pdf: issue.pdf,
  }))
  return { data: editions.sort((a, b) => b.date.localeCompare(a.date) || b.number.localeCompare(a.number, undefined, { numeric: true })), error: result.error }
}
