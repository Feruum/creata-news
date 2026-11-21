import { expect, test } from 'bun:test'
import { buildQuery, readingMinutes, publicArticles, safeExternalUrl } from './content'
test('public lists exclude unpublished and untranslated records', () => {
  const article = {status: 'published', title: {ru:'Заголовок',kk:'',en:'Title'}}
  expect(publicArticles([article, {...article,status:'review'}], 'en')).toHaveLength(1)
  expect(publicArticles([article], 'kk')).toHaveLength(0)
})
test('query serialization preserves unicode and omits empty filters', () => {
  expect(buildQuery({locale:'kk',q:'Ғылым & AI',region:''})).toBe('?locale=kk&q=%D2%92%D1%8B%D0%BB%D1%8B%D0%BC+%26+AI')
})
test('reading time and links handle invalid content safely', () => {
  expect(readingMinutes('')).toBe(1); expect(readingMinutes(Array(440).fill('word').join(' '))).toBe(2)
  expect(safeExternalUrl('javascript:alert(1)')).toBe(null); expect(safeExternalUrl('https://example.com/')).toBe('https://example.com/')
})
