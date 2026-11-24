import { expect, test } from 'bun:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import type { Article } from '@/lib/models'
import { ArticleView } from './article-view'

const campus = {
  url: 'https://upload.wikimedia.org/wikipedia/commons/a/ab/Campus.jpg',
  caption: 'Campus courtyard',
  credit: 'Test photographer',
  rights: 'CC BY 4.0',
  sourceUrl: 'https://commons.wikimedia.org/wiki/File:Campus.jpg',
  licenseUrl: 'https://creativecommons.org/licenses/by/4.0/',
}
const garden = { ...campus, url: 'https://upload.wikimedia.org/wikipedia/commons/a/aa/Garden.jpg', caption: 'Botanical garden' }
const prose = 'The university opened the campus to visitors.'

test('the website draft preview renders automatic cover and body photos with their attribution', () => {
  const emptyText = { ru: '', kk: '', en: '' }
  const article: Article = {
    id: 'campus', slug: 'campus', categoryId: 'campus', region: 'KZ', authorId: 'staff', layer: 'world',
    createdAt: '2026-11-23T00:00:00Z', updatedAt: '2026-11-23T00:00:00Z', status: 'review', revision: 1,
    title: { ...emptyText, en: 'Campus opening' }, lead: emptyText, body: { ...emptyText, en: prose },
    summary: emptyText, actions: emptyText, tags: [], sources: [], reviewNotes: [], photos: [campus, garden],
  }
  const html = renderToStaticMarkup(createElement(ArticleView, { article, locale: 'en', preview: true }))

  expect(html).toContain('src="' + campus.url + '"')
  expect(html).toContain('src="' + garden.url + '"')
  expect(html.indexOf('<figcaption>Campus courtyard')).toBeLessThan(html.indexOf(prose))
  expect(html.indexOf('<figcaption>Botanical garden')).toBeGreaterThan(html.indexOf(prose))
  expect(html).toContain('Test photographer')
  expect(html).toContain('href="' + campus.sourceUrl + '"')
  expect(html).toContain('href="' + campus.licenseUrl + '"')
  expect(html).toContain('referrerPolicy="no-referrer"')
})
