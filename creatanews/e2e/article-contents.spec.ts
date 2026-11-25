import { expect, test } from '@playwright/test'

test('article contents link to the sections on desktop and mobile', async ({ page }) => {
  const article = '/ru/articles/akku-vtoraya-zhizn'
  const contents = page.getByRole('navigation', { name: 'В этой статье' })

  for (const width of [1024, 390]) {
    await page.setViewportSize({ width, height: 800 })
    await page.goto(article, { waitUntil: 'domcontentloaded' })
    const firstSection = contents.getByRole('link', { name: 'Что это за место?', exact: true })
    await expect(firstSection).toBeVisible()
    await expect(contents.getByRole('link', { name: 'Одна деталь из воспоминаний', exact: true })).toBeVisible()
    const outline = await contents.boundingBox()
    const text = await page.locator('.article-prose').boundingBox()
    if (!outline || !text) throw new Error('Article outline or text is missing')
    if (width > 850) {
      expect(outline.x + outline.width).toBeLessThan(text.x)
    } else {
      expect(outline.y + outline.height).toBeLessThan(text.y)
    }
    const architecture = contents.getByRole('link', { name: 'Архитектурный взгляд', exact: true })
    // The first body section can reach the reading line even without a related-story feed below it.
    const firstSectionId = await firstSection.getAttribute('href')
    if (!firstSectionId) throw new Error('Section link is missing a target')
    await page.locator(firstSectionId).evaluate(section => {
      window.scrollTo({ top: scrollY + section.getBoundingClientRect().top - 50, behavior: 'instant' })
    })
    await expect(firstSection).toHaveAttribute('aria-current', 'location')
    expect(new URL(page.url()).hash).toBe('')
    await architecture.click()
    await expect(page).toHaveURL(/#article-section-/)
    await expect(page.locator(':target')).toContainText('Архитектурный взгляд:')
    await expect(page.locator(':target')).toBeInViewport()
    await expect(architecture).toHaveAttribute('aria-current', 'location')
    const sourcesLink = contents.getByRole('link', { name: 'Источники', exact: true })
    await sourcesLink.click()
    await expect(page.locator(':target').getByRole('heading', { name: 'Источники' })).toBeInViewport()
    await expect(sourcesLink).toHaveAttribute('aria-current', 'location')
    const source = page.locator(':target').getByRole('link', { name: /Zakon\.kz/ })
    await expect(source.locator('time')).toHaveText('29 июня 2024 г.')
    const dateGap = await source.evaluate(link => {
      const title = link.querySelector('strong')
      const date = link.querySelector('time')
      if (!title || !date) throw new Error('Source name or date is missing')
      return date.getBoundingClientRect().top - title.getBoundingClientRect().bottom
    })
    expect(dateGap).toBeGreaterThanOrEqual(0)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  }
})

test('segmented reading progress follows scroll and returns to the top', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  for (const width of [1280, 768]) {
    await page.setViewportSize({ width, height: 800 })
    await page.goto('/ru/articles/akku-vtoraya-zhizn')

    const progress = page.getByRole('button', { name: /^Прогресс чтения/ })
    const segments = progress.locator(':scope > span')
    await expect(segments).toHaveCount(4)
    await expect(progress).toHaveAccessibleName('Прогресс чтения: 0%. Вернуться к началу статьи')
    await expect(progress).toBeInViewport({ ratio: 1 })

    await page.evaluate(() => {
      const scrollable = document.documentElement.scrollHeight - innerHeight
      window.scrollTo({ top: scrollable * 0.12, behavior: 'instant' })
    })
    await expect(progress).toHaveText('12%')
    await expect(segments.first()).toHaveCSS('--segment-progress', '48%')
    await expect(segments.first()).toHaveCSS('background-image', /rgb\(75, 68, 246\)/)
    await expect(segments.last()).toHaveCSS('--segment-progress', '0%')
    await expect(progress).toBeInViewport({ ratio: 1 })

    await page.evaluate(() => window.scrollTo({ top: document.documentElement.scrollHeight, behavior: 'instant' }))
    await expect(progress).toHaveText('100%')
    await expect(segments.last()).toHaveCSS('--segment-progress', '100%')
    if (width === 768) {
      const nav = page.locator('.bottom-nav')
      await expect(nav).toBeInViewport()
      const [navBox, progressBox] = await Promise.all([nav.boundingBox(), progress.boundingBox()])
      if (!navBox || !progressBox) throw new Error('Bottom navigation or progress indicator is missing')
      expect(progressBox.width).toBeLessThanOrEqual(252)
      expect(progressBox.height).toBeGreaterThanOrEqual(44)
      expect(progressBox.y + progressBox.height).toBeLessThanOrEqual(navBox.y)
      await progress.click()
    } else {
      await progress.press('Enter')
    }
    await expect(progress).toHaveText('0%')
    await expect.poll(() => page.evaluate(() => scrollY)).toBe(0)
  }
})

test('phones keep the bottom navigation without the reading progress bar', async ({ browser }) => {
  const phone = await browser.newContext({ isMobile: true, hasTouch: true, viewport: { width: 320, height: 780 } })
  try {
    const page = await phone.newPage()
    await page.goto('/ru/articles/akku-vtoraya-zhizn')
    const progress = page.locator('button[aria-label^="Прогресс чтения"]')
    const nav = page.locator('.bottom-nav')

    for (const [width, height] of [[320, 780], [390, 844], [844, 390]]) {
      await page.setViewportSize({ width, height })
      await expect(nav).toBeInViewport()
      await expect(progress).toBeHidden()
      await page.evaluate(() => window.scrollTo({ top: 600, behavior: 'instant' }))
      await expect(progress).toBeHidden()
    }
    await page.setViewportSize({ width: 768, height: 1024 })
    await expect(progress).toBeInViewport()
  } finally {
    await phone.close()
  }
})
