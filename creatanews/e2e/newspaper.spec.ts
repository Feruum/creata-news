import { expect, test } from '@playwright/test'

test('newspaper menu opens an archive and the complete server-rendered edition', async ({ page, request }) => {
  await page.goto('/ru')
  await page.getByRole('button', { name: 'Открыть меню' }).click()
  await page.getByRole('dialog').getByRole('link', { name: 'Газета', exact: true }).click()
  await expect(page).toHaveURL(/\/ru\/gazeta$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Газета.')
  const original = page.locator('.gazeta-edition').filter({ hasText: '№01' })
  await expect(original.locator('.gazeta-cover img')).toBeVisible()
  await original.getByRole('link', { name: 'Читать выпуск', exact: true }).click()
  await expect(page).toHaveURL(/\/ru\/gazeta\/01$/)
  await expect(page.locator('.sheet')).toHaveCount(36)
  await expect(page.getByRole('heading', { name: 'Creata News', exact: true })).toBeVisible()
  await expect(page.locator('.lead-h').first()).toHaveText('Три дня, когда геймдев собрался в одном зале')
  const html = await (await request.get('/ru/gazeta/01')).text()
  expect(html).toContain('Сентябрь и октябрь в цифрах')
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', /\/ru\/gazeta\/01$/)
  await page.getByRole('link', { name: 'Все выпуски', exact: true }).click()
  await expect(page).toHaveURL(/\/ru\/gazeta$/)
})

test('photographs respond to keyboard and stop when reduced motion is enabled', async ({ page }) => {
  await page.goto('/ru/gazeta/01')
  const photo = page.locator('.live').first()
  await expect(photo).toHaveAttribute('role', 'button')
  await photo.focus()
  await expect(photo).toHaveAttribute('aria-pressed', 'true')
  await photo.press('Space')
  await expect(photo).toHaveAttribute('aria-pressed', 'false')
  await page.locator('#wake-all').click()
  await expect(page.locator('#wake-all')).toHaveAttribute('aria-pressed', 'true')
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await expect(page.locator('#wake-all')).toBeHidden()
  await expect(page.locator('.is-awake')).toHaveCount(0)
  expect(await page.locator('video').evaluateAll(videos => videos.every(video => (video as HTMLVideoElement).paused))).toBe(true)
})

test('newspaper adapts to mobile and tablet without leaking its layout into the archive', async ({ page }) => {
  for (const width of [360, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    await page.goto('/ru/gazeta/01')
    await expect(page.locator('.sheet').first()).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
  }
  await page.getByRole('link', { name: 'Все выпуски', exact: true }).click()
  await expect(page.locator('.site-header')).toHaveCount(1)
  await expect(page.locator('.desktop-nav a')).toHaveCount(3)
  await expect(page.locator('.gazeta-cover img').first()).toBeVisible()
})

test('issue archive has three interface languages, a legacy URL and real 404s', async ({ page, request }) => {
  for (const [locale, title] of [['ru', 'Газета.'], ['kk', 'Газет.'], ['en', 'Newspaper.']]) {
    await page.goto(`/${locale}/gazeta`)
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(title)
  }
  await page.goto('/issues/01')
  await expect(page).toHaveURL(/\/en\/gazeta\/01$/)
  await expect(page.locator('.newspaper-issue')).toHaveAttribute('lang', 'ru')
  const missing = await request.get('/ru/gazeta/999')
  // Runtime editions use the API rather than a build-time URL allowlist. Next
  // can stream its not-found UI with HTTP 200; it always marks it noindex.
  expect(await missing.text()).toContain('name="robots" content="noindex"')
})
