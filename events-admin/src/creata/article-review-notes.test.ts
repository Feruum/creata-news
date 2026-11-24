import { expect, test } from 'bun:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { ArticleReviewNotes } from './article-review-notes'

const singleSourceNote = 'Single source: editorial verification required / Один источник / Бір дереккөз'

test('review keeps the source warning visible and puts each distinct detail in a closed disclosure', () => {
  const comment = 'Уточните, какая версия вышла.'
  const html = renderToStaticMarkup(createElement(ArticleReviewNotes, {
    notes: [singleSourceNote, comment, ' Проверьте дату. ', comment, ''],
    locale: 'ru',
  }))

  expect(html).toMatch(/<details(?:\s[^>]*)?>/)
  expect(html).not.toMatch(/<details[^>]*\bopen(?:=|\s|>)/)
  expect(html.match(/<summary[\s\S]*?<\/summary>/)?.[0]).toContain('Один источник. Нужна сверка.')
  expect(html).not.toContain('editorial verification required')
  expect(html.match(/<li>/g)).toHaveLength(2)
  expect(html).toContain(`<li>${comment}</li>`)
  expect(html).toContain('<li>Проверьте дату.</li>')
})

test('review shows a source-only warning without an empty disclosure', () => {
  const html = renderToStaticMarkup(createElement(ArticleReviewNotes, {
    notes: [singleSourceNote, ' '],
    locale: 'en',
  }))

  expect(html).toContain('One source. Cross-check the facts.')
  expect(html).not.toContain('<details')
  expect(html).not.toContain('panel-description')
})
