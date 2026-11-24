import { safeExternalUrl } from '@creata/contracts'
import { localized, storyUpdateDateInput } from './domain'
import { tr, type CopyKey } from './i18n'
import type { Article, Locale } from './types'

export const revisionFields = {
  body: 'body', lead: 'lead', summary: 'summary', actions: 'actions',
  actionGuide: 'storyGuide', storyUpdates: 'storyUpdates',
} as const satisfies Record<string, CopyKey>

export type RevisionField = keyof typeof revisionFields
export type RevisionLine = { text: string; href?: string; updateId?: string }

export function revisionLines(article: Pick<Article, RevisionField>, field: RevisionField, language: Locale, locale: Locale): RevisionLine[] {
  const line = (label: CopyKey, value?: string): RevisionLine => ({ text: `${tr(label, locale)} · ${value || '—'}` })
  const source = (label: CopyKey, value: string): RevisionLine => {
    const href = safeExternalUrl(value)
    const result = line(label, value)
    return href ? { ...result, href } : result
  }

  if (field === 'actionGuide') {
    const guide = article.actionGuide
    return guide ? [
      line('storyGuideAudience', guide.audience[language]),
      line('storyGuideSteps', guide.steps[language]),
      source('storyGuideSource', guide.sourceUrl),
    ] : []
  }

  if (field === 'storyUpdates') {
    const updates = [...(article.storyUpdates ?? [])]
      .sort((left, right) => Date.parse(right.occurredAt) - Date.parse(left.occurredAt))

    return updates.flatMap((update, index) => {
      const timestamp = storyUpdateDateInput(update.occurredAt).replace('T', ' ').replace(/\.000$/, '')
      const rows = [
        line('storyUpdateDate', timestamp),
        line('storyUpdateTitle', update.title[language]),
        line('storyUpdateBody', update.body[language]),
        source('storyUpdateSource', update.sourceUrl),
      ]
      if (update.articleSlug) rows.push(line('storyUpdateArticle', update.articleSlug))
      return [
        ...(index ? [{ text: '' }] : []),
        ...rows.map(row => ({ ...row, updateId: update.id })),
      ]
    })
  }

  return localized(article[field], language).split('\n').map(text => ({ text }))
}
