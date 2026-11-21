import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      headers: {
        'access-control-allow-origin': new URL(page.url()).origin,
        'access-control-allow-credentials': 'true'
      },
      json: {
        id: 'analytics-session',
        active: true,
        expires_at: '2099-01-01T00:00:00Z',
        identity: { id: 'analyst', traits: { email: 'analyst@test.local' } }
      }
    })
  )
})

test('menu opens real analytics and displays API values and audience', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-analytics.html?home')
  await page.getByRole('link', { name: 'Analytics', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Sunday Open · Analytics' })).toBeVisible()
  await expect(page.getByText('EMS: 4 · внешние: 2')).toBeVisible()
  await expect(page.getByText('50% активных регистраций')).toBeVisible()
  await expect(page.getByText('Live API workshop')).toBeVisible()
  await expect(page.getByText('2 из 8 аккаунтов')).toBeVisible()
  await expect(page.getByText('MOCK DATA')).toHaveCount(0)
  await page.keyboard.press('Control+k')
  await expect(page.getByRole('dialog').getByText('Analytics', { exact: true })).toBeVisible()
  await page.keyboard.press('Escape')
})

test('loads, retries errors, handles empty and denied responses', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-analytics.html?slow&error')
  await expect(page.getByRole('status', { name: 'Загружаем аналитику Sunday Open' })).toBeVisible()
  await expect(page.getByText('Не удалось загрузить аналитику Sunday Open')).toBeVisible()
  await page.getByRole('button', { name: 'Повторить', exact: true }).click()
  await expect(page.getByText('Live API workshop')).toBeVisible()
  await page.goto('/e2e/fixtures/sunday-analytics.html?empty')
  await expect(page.getByText('Пока нет событий Sunday Open')).toBeVisible()
  await expect(page.getByText('0% активных регистраций')).toBeVisible()
  await page.goto('/e2e/fixtures/sunday-analytics.html?denied')
  await expect(page.getByText('Нет доступа к аналитике Sunday Open')).toBeVisible()
})

test('polls check-ins and supports manual refresh', async ({ page }) => {
  await page.clock.install()
  await page.goto('/e2e/fixtures/sunday-analytics.html')
  await expect(page.getByText('50% активных регистраций')).toBeVisible()
  await page.evaluate(() => window.dispatchEvent(new Event('check-in')))
  await page.clock.fastForward(16000)
  await expect(page.getByText('67% активных регистраций')).toBeVisible()
  await page.getByRole('button', { name: 'Обновить', exact: true }).first().click()
  await expect.poll(() => page.locator('html').getAttribute('data-analytics-requests')).toBe('3')
  await page.evaluate(() => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
  })
  await page.clock.fastForward(30000)
  await expect(page.locator('html')).toHaveAttribute('data-analytics-requests', '3')
  await page.evaluate(() => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
  })
  await expect.poll(() => page.locator('html').getAttribute('data-analytics-requests')).toBe('4')
})

test('regular users get no menu entry and make no analytics request', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-analytics.html?regular')
  await expect(page.getByRole('heading', { name: 'Доступ к статистике ограничен' })).toBeVisible()
  await expect(page.locator('a[href="/analytics"]')).toHaveCount(0)
  await page.keyboard.press('Control+k')
  await expect(page.getByRole('dialog').getByText('Analytics', { exact: true })).toHaveCount(0)
  await expect(page.locator('html')).not.toHaveAttribute('data-analytics-requests')
  await page.goto('/e2e/fixtures/sunday-analytics.html?regular&dashboard')
  await expect(page.getByRole('heading', { name: 'Доступ к статистике ограничен' })).toBeVisible()
  await expect(page.locator('html')).not.toHaveAttribute('data-analytics-requests')
})

test('administrators can open analytics and the mobile summary fits the viewport', async ({ page }, testInfo) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.goto('/e2e/fixtures/sunday-analytics.html?admin&home')
  await page.getByRole('link', { name: 'Analytics', exact: true }).click()
  const summary = page.locator('section[aria-labelledby="sunday-open-analytics-title"]')
  await expect(summary.getByText('Live API workshop')).toBeVisible()
  await summary.screenshot({ path: testInfo.outputPath('sunday-analytics-desktop.png') })
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(summary.getByText('EMS: 4 · внешние: 2')).toBeVisible()
  const bounds = await summary.boundingBox()
  expect(bounds).not.toBeNull()
  expect(bounds!.x).toBeGreaterThanOrEqual(0)
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390)
  expect(await summary.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true)
  await summary.screenshot({ path: testInfo.outputPath('sunday-analytics-mobile.png') })
  expect(errors).toEqual([])
})
