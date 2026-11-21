import { expect, test } from '@playwright/test'

test.setTimeout(60_000)
for (const width of [390, 1280]) {
  test(`heatmap includes every day across year and theme changes at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await page.clock.setFixedTime(new Date('2026-10-28T12:00:00Z'))
    await page.goto('/e2e/fixtures/activity-dashboard.html')
    const cells = page.locator('rect[data-date]')
    await expect(cells).toHaveCount(365)
    expect(
      await cells.evaluateAll((rects) =>
        rects.every((rect) => getComputedStyle(rect).opacity === '1' && getComputedStyle(rect).fill !== 'rgb(0, 0, 0)')
      )
    ).toBe(true)
    await expect(page.locator('rect[data-date="2026-01-01"]')).toHaveCount(1)
    const calendar = page.getByRole('region', { name: /Event activity for/ })
    await calendar.evaluate((element) => {
      element.scrollLeft = element.scrollWidth
    })
    await expect(page.locator('rect[data-date="2026-12-31"]')).toBeInViewport()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await page.getByRole('button', { name: '2024', exact: true }).click()
    await expect(cells).toHaveCount(366)
    await expect(page.locator('rect[data-date="2024-02-29"]')).toHaveAttribute('data-count', '0')
    await expect(page.getByText('0 events in 2024')).toBeVisible()
    await page.getByRole('button', { name: 'Dark theme', exact: true }).click()
    await expect(cells.first()).toHaveCSS('fill', 'rgb(40, 51, 66)')
    expect(await cells.evaluateAll((rects) => rects.every((rect) => getComputedStyle(rect).opacity === '1'))).toBe(true)
  })
}

test('team feed, entity timeline and full log distinguish users from system actions', async ({ page }) => {
  await page.goto('/e2e/fixtures/activity-dashboard.html')
  await expect(page.getByText('Ada Initiator', { exact: true })).toHaveCount(2)
  await expect(page.getByText('User #8', { exact: true })).toHaveCount(2)
  await expect(page.getByText('System', { exact: true })).toHaveCount(2)
  await page.getByRole('button', { name: 'See More' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog.getByText('Ada Initiator', { exact: true })).toBeVisible()
  await expect(dialog.getByText('User #8', { exact: true })).toBeVisible()
  await expect(dialog.getByText('System', { exact: true })).toBeVisible()
})
