import type { Locale, Localized } from './models'
export function buildQuery(params: Record<string, string | undefined>) { const query = new URLSearchParams(Object.entries(params).filter((entry): entry is [string,string] => !!entry[1])); return query.size ? `?${query}` : '' }
export function readingMinutes(body: string) { return Math.max(1, Math.ceil(body.trim().split(/\s+/).filter(Boolean).length / 220)) }
export function publicArticles<T extends { status: string; title: Localized }>(articles: T[], locale: Locale): T[] { return articles.filter(article => article.status === 'published' && !!article.title?.[locale]?.trim()) }
export function safeExternalUrl(value: string): string | null { try { const url = new URL(value); return ['http:','https:'].includes(url.protocol) && !url.username && !url.password ? url.toString() : null } catch { return null } }
