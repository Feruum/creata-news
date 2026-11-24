import type { Article, Locale, Localized } from './models'
import { localize } from './i18n'
export { articleBodyBlocks, safeExternalUrl } from '@creata/contracts'
export function buildQuery(params: Record<string, string | undefined>) { const query = new URLSearchParams(Object.entries(params).filter((entry): entry is [string,string] => !!entry[1])); return query.size ? `?${query}` : '' }
export function readingMinutes(body: string) { return Math.max(1, Math.ceil(body.trim().split(/\s+/).filter(Boolean).length / 220)) }
export function articleReadingMinutes(article: Article, locale: Locale) {
  return article.readingMinutes?.[locale] ?? readingMinutes(localize(article.body, locale))
}
export function publicArticles<T extends { status: string; title: Localized }>(articles: T[], locale: Locale): T[] { return articles.filter(article => article.status === 'published' && !!article.title?.[locale]?.trim()) }
