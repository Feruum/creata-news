import type { Article, Locale } from './models'
import { safeExternalUrl } from './content'
import { formatDate, localize } from './i18n'

export function localizedStoryUpdates(updates: Article['storyUpdates'], locale: Locale) {
  return (updates ?? []).flatMap(update => {
    const title = localize(update.title, locale).trim()
    const body = localize(update.body, locale).trim()
    const sourceUrl = safeExternalUrl(update.sourceUrl)
    return title && body && sourceUrl && Number.isFinite(Date.parse(update.occurredAt))
      ? [{ ...update, title, body, sourceUrl }]
      : []
  }).sort((left, right) => Date.parse(right.occurredAt) - Date.parse(left.occurredAt))
}

export function followedStorySummary(updates: Article['storyUpdates'], unreadUpdateIds: readonly string[], locale: Locale) {
  const available = localizedStoryUpdates(updates, locale)
  const latest = available[0]
  if (!latest) return null
  const unread = available.filter(update => unreadUpdateIds.includes(update.id))
  return { latest, target: unread[0] ?? latest, unreadCount: unread.length }
}

export function localizedActionGuide(guide: Article['actionGuide'], locale: Locale) {
  if (!guide) return null
  const audience = localize(guide.audience, locale).trim()
  const steps = localize(guide.steps, locale).trim()
  const sourceUrl = safeExternalUrl(guide.sourceUrl)
  return audience && steps && sourceUrl ? { audience, steps, sourceUrl } : null
}

const timeZone = 'Asia/Almaty'
const storyClock = new Intl.DateTimeFormat('en-GB', {
  timeZone,
  hour: '2-digit',
  minute: '2-digit',
  timeZoneName: 'shortOffset',
})

export function formatStoryTime(timestamp: string, locale: Locale) {
  return `${formatDate(timestamp, locale, false, true, timeZone)} · ${storyClock.format(new Date(timestamp))}`
}
