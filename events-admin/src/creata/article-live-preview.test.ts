import { expect, test } from 'bun:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { ArticleLivePreview } from './article-live-preview'
import { ArticleCoverPreview, ArticlePhotosPanel } from './article-photos'
import { emptyText } from './domain'
import type { Article } from './types'

const campus = {
  url: 'https://upload.wikimedia.org/wikipedia/commons/a/ab/Campus.jpg',
  caption: 'Campus courtyard',
  credit: 'Test photographer',
  rights: 'CC BY 4.0',
  sourceUrl: 'https://commons.wikimedia.org/wiki/File:Campus.jpg',
  licenseUrl: 'https://creativecommons.org/licenses/by/4.0/',
}
const garden = { ...campus, url: 'https://upload.wikimedia.org/wikipedia/commons/a/aa/Garden.jpg', caption: 'Botanical garden' }
const manualCover = { url: '/media/cover.jpg', caption: 'Manual cover', credit: 'Newsroom', rights: 'Original' }
const prose = 'The university opened the campus to visitors.'

function renderPhotos(photos: NonNullable<Article['photos']>, media?: Article['media']) {
  const article = { media, photos, body: { ...emptyText(), en: prose } } as Article
  return renderToStaticMarkup(createElement(ArticleLivePreview, {
    article, language: 'en', label: 'Live preview', publicWebsite: 'https://example.org',
  }))
}

test('live preview uses the first Wikimedia photo as the cover and credits the photos', () => {
  const html = renderPhotos([campus, garden])
  expect(html.indexOf('<figure')).toBeLessThan(html.indexOf(prose))
  expect(html.indexOf('<figcaption>Campus courtyard')).toBeLessThan(html.indexOf(prose))
  expect(html.indexOf('<figcaption>Botanical garden')).toBeGreaterThan(html.indexOf(prose))
  expect(html.match(/<figure/g)).toHaveLength(2)
  expect(html).toContain('Test photographer')
  expect(html).toContain('CC BY 4.0')
  expect(html).toContain(`href="${campus.sourceUrl}"`)
  expect(html).toContain(`href="${campus.licenseUrl}"`)
  expect(html).toContain('referrerPolicy="no-referrer"')
})

test('live preview keeps Wikimedia photos within the article when a manual cover exists', () => {
  const html = renderPhotos([campus, garden], manualCover)
  expect(html).toContain('src="/media/cover.jpg"')
  expect(html.indexOf('<figcaption>Manual cover')).toBeLessThan(html.indexOf(prose))
  expect(html.indexOf('<figcaption>Campus courtyard')).toBeGreaterThan(html.indexOf(prose))
  expect(html.indexOf('<figcaption>Botanical garden')).toBeGreaterThan(html.indexOf(prose))
})

test('live preview skips untrusted image URLs and unsafe attribution links', () => {
  const html = renderPhotos([
    { ...campus, url: 'https://upload.wikimedia.org.example.com/wikipedia/commons/Campus.jpg', caption: 'Untrusted photo' },
    { ...garden, sourceUrl: 'javascript:alert(1)', licenseUrl: 'data:text/html,test' },
  ])
  expect(html).not.toContain('Untrusted photo')
  expect(html).not.toContain('upload.wikimedia.org.example.com')
  expect(html).not.toContain('javascript:')
  expect(html).not.toContain('data:text/html')
  expect(html).toContain('<figcaption>Botanical garden')
  expect(html).toContain('CC BY 4.0')
})

test('the Hermes photo list keeps the cover removal control without repeating the cover image', () => {
  const html = renderToStaticMarkup(createElement(ArticlePhotosPanel, {
    photos: [campus, garden], cover: campus, language: 'en', disabled: false,
    onChange: () => undefined,
  }))
  expect(html).not.toContain(`src="${campus.url}"`)
  expect(html).toContain('Cover image · Campus courtyard')
  expect(html).toContain('aria-label="Remove photo · Campus courtyard"')
  expect(html).toContain(`src="${garden.url}"`)
})

test('the Media cover preview shows a Hermes cover and its attribution in the original cover slot', () => {
  const emptyMedia = { url: '', caption: '', credit: '', rights: '' }
  const renderCover = (media: Article['media'] = emptyMedia, photo?: typeof campus) =>
    renderToStaticMarkup(createElement(ArticleCoverPreview, { media, photo, language: 'en' }))

  const automatic = renderCover(emptyMedia, campus)
  expect(automatic).toContain(`src="${campus.url}"`)
  expect(automatic).toContain('Campus courtyard · Test photographer')
  expect(automatic).toContain(`href="${campus.licenseUrl}"`)
  expect(automatic).not.toContain('cover-placeholder')
  expect(renderCover(manualCover, campus)).toContain('src="/media/cover.jpg"')
  expect(renderCover(manualCover, campus)).not.toContain(campus.url)
  expect(renderCover()).toContain('cover-placeholder')
})
