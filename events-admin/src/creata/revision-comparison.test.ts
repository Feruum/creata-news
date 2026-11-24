import { expect, test } from 'bun:test'
import type { Article } from './types'
import { emptyText } from './domain'
import { revisionLines } from './revision-comparison'

const revision = {
  body: { ...emptyText(), en: 'Lead paragraph.\n\nSecond paragraph.' },
  lead: emptyText(), summary: emptyText(), actions: emptyText(),
  actionGuide: {
    audience: { ...emptyText(), ru: 'Студенты', en: 'Students' },
    steps: { ...emptyText(), en: 'Apply online' }, sourceUrl: 'https://example.org/guide',
  },
  storyUpdates: [{
    id: 'new', occurredAt: '2026-11-14T03:20:07+05:00',
    title: { ...emptyText(), ru: 'Отбор', en: 'Selection' },
    body: { ...emptyText(), en: 'The teams were selected.' },
    sourceUrl: 'https://example.org/results', articleSlug: 'команды',
  }],
} satisfies Pick<Article, 'body' | 'lead' | 'summary' | 'actions' | 'actionGuide' | 'storyUpdates'>

test('revision comparison keeps the existing text diff lines', () => {
  expect(revisionLines(revision, 'body', 'en', 'ru')).toEqual([
    { text: 'Lead paragraph.' }, { text: '' }, { text: 'Second paragraph.' },
  ])
})

test('revision comparison shows the selected language, UTC date and source links', () => {
  const guide = revisionLines(revision, 'actionGuide', 'en', 'ru')
  expect(guide).toContainEqual({ text: 'Для кого · Students' })
  expect(guide).toContainEqual({ text: 'Источник инструкции · https://example.org/guide', href: 'https://example.org/guide' })
  expect(revisionLines(revision, 'actionGuide', 'kk', 'ru')).toContainEqual({ text: 'Для кого · —' })

  const updates = revisionLines(revision, 'storyUpdates', 'en', 'ru')
  expect(updates).toContainEqual(expect.objectContaining({ text: 'Дата и время обновления (UTC) · 2026-11-13 22:20:07', updateId: 'new' }))
  expect(updates).toContainEqual(expect.objectContaining({ text: 'Заголовок обновления · Selection', updateId: 'new' }))
  expect(updates).toContainEqual(expect.objectContaining({ text: 'Источник обновления · https://example.org/results', href: 'https://example.org/results' }))
  expect(updates).toContainEqual(expect.objectContaining({ text: 'Слаг связанного материала · команды' }))
  expect(revisionLines({ ...revision, storyUpdates: [] }, 'storyUpdates', 'en', 'ru')).toEqual([])
})
