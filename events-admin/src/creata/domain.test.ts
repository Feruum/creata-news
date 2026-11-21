import { describe, expect, test } from 'bun:test'
import { allowedArticleActions, emptyText, filterArticles, localized, validateSource } from './domain'
import type { Article, User } from './types'

const user = (roles: string[], verticals: string[] = []): User => ({ id: 'u1', name: 'Editor', locale: 'ru', stage: 'exploring', topics: [], frequency: 'daily', paused: false, roles, verticals, telegramLinked: false })
const article = { id: 'a1', authorId: 'u1', categoryId: 'ai', layer: 'world', status: 'review', revision: 4, title: { ru: 'Новость', kk: 'Жаңалық', en: 'News' }, region: 'KZ', tags: [], sources: [] } as unknown as Article

describe('editorial authorization affordances', () => {
  test('authors submit their own revision but cannot approve or publish it', () => {
    expect(allowedArticleActions(user(['author']), article)).toEqual(['submit'])
    expect(allowedArticleActions(user(['author']), { ...article, authorId: 'other' })).toEqual([])
  })
  test('editor publication respects assigned vertical and ecosystem chief approval', () => {
    expect(allowedArticleActions(user(['editor'], ['science']), article)).toEqual([])
    expect(allowedArticleActions(user(['editor'], ['ai']), article)).toContain('approve')
    expect(allowedArticleActions(user(['editor'], ['ai']), { ...article, layer: 'eco', status: 'approved', approvedRevision: 4 })).not.toContain('publish')
    expect(allowedArticleActions(user(['editor'], ['ai']), { ...article, layer: 'eco' })).not.toContain('approve')
    expect(allowedArticleActions(user(['editor'], ['ai']), { ...article, layer: 'eco' })).not.toContain('reject')
    expect(allowedArticleActions(user(['chief_editor']), { ...article, layer: 'eco', status: 'approved', approvedRevision: 4 })).toContain('publish')
  })
  test('publishing requires approval of the current revision', () => {
    expect(allowedArticleActions(user(['chief_editor']), { ...article, status: 'approved', approvedRevision: 3 })).not.toContain('publish')
    expect(allowedArticleActions(user(['chief_editor']), { ...article, status: 'approved', approvedRevision: 4 })).toContain('schedule')
  })
})
describe('shared display and forms', () => {
  test('locale fallbacks preserve source text without inventing translated content', () => {
    expect(localized({ ru: 'Источник', kk: '', en: 'Source' }, 'kk')).toBe('Источник')
    const a = emptyText(); a.ru = 'one'; expect(emptyText().ru).toBe('')
  })
  test('article search combines title, author and filters', () => {
    expect(filterArticles([article], { query: 'жаңа', locale: 'kk', status: '', categoryId: '', authorId: '' })).toHaveLength(1)
    expect(filterArticles([article], { query: '', locale: 'ru', status: 'published', categoryId: '', authorId: '' })).toHaveLength(0)
  })
  test('sources require a safe scheme and reasonable collection interval', () => {
    expect(validateSource({ name: 'Feed', type: 'rss', url: 'javascript:alert(1)', intervalMinutes: 30 })).toBe('invalidUrl')
    expect(validateSource({ name: 'Channel', type: 'telegram', url: 'https://t.me/creata_news', intervalMinutes: 15 })).toBeNull()
    expect(validateSource({ name: 'Feed', type: 'rss', url: 'https://example.com/feed', intervalMinutes: 0 })).toBe('invalidInterval')
  })
  test('language and date filters use translated content and the campus calendar day', () => {
    const item = {...article, title: {...article.title, en: ''}, updatedAt: '2026-11-13T20:00:00Z'}
    const base = {query:'',locale:'ru' as const,status:'',categoryId:'',authorId:''}
    expect(filterArticles([item], {...base, language:'en'})).toHaveLength(0)
    expect(filterArticles([item], {...base, from:'2026-11-14', to:'2026-11-14'})).toHaveLength(1)
    expect(filterArticles([item], {...base, to:'2026-11-13'})).toHaveLength(0)
  })
})
