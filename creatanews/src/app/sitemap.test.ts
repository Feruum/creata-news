import { expect, spyOn, test } from 'bun:test'
import sitemap from './sitemap'
import { siteOrigin } from '@/lib/api'

test('the sitemap lists graphics at their canonical chart URLs', async () => {
  const title = { ru: 'Материал', kk: 'Мақала', en: 'Story' }
  const articles = [
    { slug: 'news-story', format: 'article', status: 'published', title },
    { slug: 'data-story', format: 'graphic', status: 'published', title },
    { slug: 'private-data', format: 'graphic', status: 'draft', title },
  ]
  const fetchApi = spyOn(globalThis, 'fetch').mockImplementation(Object.assign(async (input: RequestInfo | URL) => {
    const url = new URL(String(input))
    return Response.json({ items: url.pathname.endsWith('/public/articles') ? articles : [] })
  }, { preconnect: fetch.preconnect }))
  try {
    const entries = await sitemap()
    const origin = siteOrigin()
    for (const locale of ['ru', 'kk', 'en']) {
      expect(entries.find(entry => entry.url === `${origin}/${locale}/charts/data-story`)?.alternates?.languages).toEqual({
        ru: `${origin}/ru/charts/data-story`,
        kk: `${origin}/kk/charts/data-story`,
        en: `${origin}/en/charts/data-story`,
      })
      expect(entries.some(entry => entry.url === `${origin}/${locale}/articles/news-story`)).toBe(true)
    }
    expect(entries.some(entry => entry.url.endsWith('/articles/data-story') || entry.url.endsWith('/private-data'))).toBe(false)
  } finally {
    fetchApi.mockRestore()
  }
})
