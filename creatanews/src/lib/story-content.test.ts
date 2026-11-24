import { expect, test } from 'bun:test'
import type { Article } from './models'
import { followedStorySummary, localizedActionGuide, localizedStoryUpdates } from './story-content'

const text = (ru: string, en = '') => ({ ru, kk: '', en })
const updates = [
  { id: 'announced', occurredAt: '2026-11-12T09:00:00+05:00', title: text('Анонс', 'Announcement'), body: text('Первый этап', 'First stage'), sourceUrl: 'https://example.org/announcement' },
  { id: 'selected', occurredAt: '2026-11-21T09:00:00+05:00', title: text('Отбор'), body: text('Названы команды'), sourceUrl: 'https://example.org/selection', articleSlug: 'команды' },
  { id: 'incomplete', occurredAt: '2026-11-22T09:00:00+05:00', title: text('Без текста'), body: text(''), sourceUrl: 'https://example.org/incomplete' },
] satisfies NonNullable<Article['storyUpdates']>

test('shows the freshest translated updates first without modifying the article', () => {
  expect(localizedStoryUpdates(updates, 'ru').map(item => item.id)).toEqual(['selected', 'announced'])
  expect(localizedStoryUpdates(updates, 'en').map(item => item.title)).toEqual(['Announcement'])
  expect(localizedStoryUpdates(undefined, 'ru')).toEqual([])
  expect(updates[0].id).toBe('announced')
})

test('a followed story links to the newest unread update available in the reader’s language', () => {
  const summary = followedStorySummary(updates, ['announced', 'incomplete', 'removed'], 'ru')
  expect(summary?.latest.id).toBe('selected')
  expect(summary?.target.id).toBe('announced')
  expect(summary?.unreadCount).toBe(1)
  expect(followedStorySummary(updates, ['selected'], 'en')?.unreadCount).toBe(0)
  expect(followedStorySummary(updates, [], 'ru')?.target.id).toBe('selected')
  expect(followedStorySummary(updates, ['announced'], 'kk')).toBeNull()
})

test('renders a practical guide only with local instructions and a safe source', () => {
  const guide = { audience: text('Студенты'), steps: text('Откройте форму'), sourceUrl: 'https://example.org/rules' }
  expect(localizedActionGuide(guide, 'ru')).toEqual({ audience: 'Студенты', steps: 'Откройте форму', sourceUrl: guide.sourceUrl })
  expect(localizedActionGuide(guide, 'kk')).toBeNull()
  expect(localizedActionGuide({ ...guide, sourceUrl: 'javascript:alert(1)' }, 'ru')).toBeNull()
})
