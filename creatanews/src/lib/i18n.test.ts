import { expect, test } from 'bun:test'
import { copy, formatDate, isLocale, localize, localePath, switchLocale } from './i18n'
test('all three locales have every interface string and Kazakh is not a Russian alias', () => {
  for (const locale of ['ru', 'kk', 'en'] as const) for (const key of Object.keys(copy.ru) as (keyof typeof copy.ru)[]) expect(copy[locale][key].trim().length).toBeGreaterThan(0)
  expect(copy.kk.latest).not.toBe(copy.ru.latest)
})
test('locale routing rejects unknown codes and retains deep links when switching', () => {
  expect(isLocale('kk')).toBe(true); expect(isLocale('kz')).toBe(false)
  expect(localePath('ru')).toBe('/ru')
  expect(localePath('kk', '/articles/story')).toBe('/kk/articles/story')
  expect(switchLocale('/ru/articles/story', 'en')).toBe('/en/articles/story')
  expect(switchLocale('/profile', 'kk')).toBe('/kk/profile')
})
test('missing translations remain visibly unavailable instead of silently changing language', () => {
  expect(localize({ru: 'Русский', kk: '', en: 'English'}, 'kk')).toBe('')
})
test('date labels are deterministic across server and browser ICU with the configured timezone', () => {
  const instant='2026-11-13T20:11:30Z'
  expect(formatDate(instant,'kk')).toBe('14 қараша 2026')
  expect(formatDate(instant,'kk',true)).toBe('14 қар.')
  expect(formatDate(instant,'ru')).toBe('14 ноября 2026 г.')
  expect(formatDate(instant,'en')).toBe('14 November 2026')
  expect(formatDate('invalid','kk')).toBe('')
  expect(formatDate(undefined,'en')).toBe('')
})
