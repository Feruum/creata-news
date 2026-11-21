import { expect, test } from '@playwright/test'

test.setTimeout(60_000)

for (const theme of ['light', 'dark']) {
  for (const width of [390, 1280]) {
    test(`activity colors, totals and ranges: ${theme}, ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 })
      await page.goto(`/e2e/fixtures/event-activity.html${theme === 'dark' ? '?dark' : ''}`)
      const totals = page.getByRole('definition')
      await expect(totals).toHaveText(['3', '15'])
      await expect(page.getByText('Most registrations:')).toContainText('11')
      const bar = page.locator('.recharts-bar-rectangle path').first()
      await expect(bar).toBeVisible()
      await expect(bar).toHaveCSS('fill', theme === 'light' ? 'rgb(77, 124, 15)' : 'rgb(163, 230, 53)')
      await expect(page.locator('.recharts-area-area')).toHaveCSS(
        'fill',
        theme === 'light' ? 'rgb(101, 163, 13)' : 'rgb(190, 242, 100)'
      )
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
      await page.getByRole('button', { name: 'Last 7 days', exact: true }).click()
      await expect(totals).toHaveText(['2', '11'])
      await page.getByRole('button', { name: 'Last 90 days', exact: true }).click()
      await expect(totals).toHaveText(['6', '21'])
    })
  }
}

test('empty activity shows zero totals and errors offer retry without misleading zeros', async ({ page }) => {
  await page.goto('/e2e/fixtures/event-activity.html?empty')
  await expect(page.getByText('No activity in this period')).toBeVisible()
  await expect(page.getByRole('definition')).toHaveText(['0', '0'])
  await page.goto('/e2e/fixtures/event-activity.html?error')
  await expect(page.getByRole('alert')).toContainText('Could not load event activity')
  await expect(page.getByRole('definition')).toHaveCount(0)
  await page.getByRole('button', { name: 'Try again' }).click()
  await expect(page.getByRole('definition')).toHaveText(['3', '15'])
})
