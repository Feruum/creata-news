import type { Article, ArticleAction, LocalizedText, Locale, Settings, Source, User } from './types'
export const emptyText = (): LocalizedText => ({ ru: '', kk: '', en: '' })
export const localized = (text: LocalizedText | string | null | undefined, locale: Locale): string => typeof text === 'string' ? text : text?.[locale] || text?.ru || text?.en || text?.kk || ''
export function canAccessStoryFollowUps(user: Pick<User, 'roles'> | null | undefined): boolean {
  return !!user?.roles.some(role => role === 'chief_editor' || role === 'editor')
}
export function canViewAudit(user: Pick<User, 'roles'> | null | undefined): boolean {
  return !!user?.roles.some(role => role === 'admin' || role === 'chief_editor')
}
export function canManageEditorialCatalog(user: Pick<User, 'roles'> | null | undefined): boolean {
  return !!user?.roles.some(role => role === 'admin' || role === 'chief_editor' || role === 'editor')
}
export function canPublishArticle(user: Pick<User, 'roles' | 'verticals'>, article: Pick<Article, 'layer' | 'categoryId'>): boolean {
  return user.roles.includes('chief_editor') ||
    (article.layer === 'world' && user.roles.includes('editor') && user.verticals.includes(article.categoryId))
}
export function allowedArticleActions(user: User, article: Article): ArticleAction[] {
  const chief = user.roles.includes('chief_editor')
  const editor = user.roles.includes('editor') && user.verticals.includes(article.categoryId)
  const owner = user.roles.includes('author') && article.authorId === user.id
  if (!chief && !editor) return owner ? ['submit'] : []
  const actions: ArticleAction[] = ['submit', 'rework']
  if (canPublishArticle(user, article)) {
    actions.push('reject')
    if (article.status === 'review' || article.status === 'rework' || article.status === 'new' || article.status === 'unpublished') actions.push('approve')
    if (article.approvedRevision === article.revision && (article.status === 'approved' || article.status === 'scheduled')) actions.push('publish', 'schedule')
    if (article.publishedRevision) actions.push('unpublish')
  }
  return actions
}
export function filterArticles(items: Article[], filters: { query: string; locale: Locale; status: string; categoryId: string; authorId: string; language?: Locale | ''; from?: string; to?: string }): Article[] {
  const q = filters.query.trim().toLocaleLowerCase()
  return items.filter(a => { const day = calendarDay(a.updatedAt || a.publishedAt || a.createdAt); return (!filters.language || !!a.title?.[filters.language]?.trim()) && (!filters.from || day >= filters.from) && (!filters.to || day <= filters.to) && (!filters.status || a.status === filters.status) && (!filters.categoryId || a.categoryId === filters.categoryId) && (!filters.authorId || a.authorId === filters.authorId) && (!q || [localized(a.title, filters.locale), a.author?.name || '', a.region, ...(a.tags || [])].join(' ').toLocaleLowerCase().includes(q)) })
}
export function validateSource(source: Pick<Source, 'name' | 'type' | 'url' | 'intervalMinutes'>): 'invalidUrl' | 'invalidInterval' | 'required' | null {
  if (!source.name.trim() || !source.url.trim()) return 'required'
  try { const url = new URL(source.url); if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || (source.type === 'telegram' && !['t.me', 'telegram.me'].includes(url.hostname))) return 'invalidUrl' } catch { return 'invalidUrl' }
  if (!Number.isInteger(source.intervalMinutes) || source.intervalMinutes < 1 || source.intervalMinutes > 1440) return 'invalidInterval'
  return null
}
export const statuses = ['new', 'writing', 'review', 'rework', 'approved', 'scheduled', 'published', 'unpublished', 'rejected'] as const
export const articleFields = (a: Article) => ({
  slug: a.slug, title: a.title, lead: a.lead, body: a.body, summary: a.summary, actions: a.actions,
  layer: a.layer, categoryId: a.categoryId, region: a.region, tags: a.tags, authorId: a.authorId,
  editorId: (a as Article & {editorId?:string}).editorId, sources: a.sources, reviewNotes: a.reviewNotes, media: a.media,
  actionGuide: a.actionGuide ?? null, storyUpdates: a.storyUpdates ?? [], photos: a.photos,
})

export function aiSettingsPatch(ai: Settings['ai'], secret: string, exa?: { key: string; clear: boolean }) {
  return {
    endpoint: ai.endpoint,
    model: ai.model,
    dailyBudgetUsd: ai.dailyBudgetUsd,
    inputPricePerMillion: ai.inputPricePerMillion,
    outputPricePerMillion: ai.outputPricePerMillion,
    apiKey: secret.trim() || undefined,
    exaApiKey: exa?.clear ? '' : exa?.key.trim() || undefined,
  }
}

export const csv = (value: string) => [...new Set(value.split(',').map(s => s.trim()).filter(Boolean))]

export function calendarDay(value: string): string { const date=new Date(value); if(Number.isNaN(date.getTime()))return '';const parts=new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Qyzylorda',year:'numeric',month:'2-digit',day:'2-digit'}).formatToParts(date);return ['year','month','day'].map(type=>parts.find(p=>p.type===type)?.value||'').join('-') }

export function storyFollowUpStatus(dueDate: string, now = new Date().toISOString()) {
  const today = calendarDay(now)
  if (dueDate < today) return 'followUpOverdue'
  return dueDate === today ? 'today' : 'followUpUpcoming'
}

export function storyUpdateDateInput(timestamp: string): string {
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? '' : date.toISOString().slice(0, -1)
}

export function storyUpdateTimestamp(input: string): string {
  const date = new Date(`${input}Z`)
  return input && !Number.isNaN(date.getTime()) ? date.toISOString() : ''
}
