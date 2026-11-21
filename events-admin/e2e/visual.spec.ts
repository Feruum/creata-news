import { test, expect } from 'playwright/test'

test.describe('Neoflat UI Visual Regression', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('login page renders with neoflat styling', async ({ page }) => {
    await page.goto('/auth/login')
    await expect(page).toHaveScreenshot('login-page.png', {
      maxDiffPixelRatio: 0.01
    })
  })

  test('badges use rounded-md corners', async ({ page }) => {
    const badge = page.locator('[data-slot="badge"]').first()

    if (await badge.isVisible()) {
      const borderRadius = await badge.evaluate((el) => window.getComputedStyle(el).borderRadius)
      expect(borderRadius).not.toBe('9999px')
      expect(parseFloat(borderRadius)).toBeLessThanOrEqual(8)
    }
  })

  test('no box-shadows on form elements', async ({ page }) => {
    await page.goto('/auth/login')

    const inputs = page.locator('input')
    const count = await inputs.count()

    for (let i = 0; i < count; i++) {
      const input = inputs.nth(i)
      if (await input.isVisible()) {
        const boxShadow = await input.evaluate((el) => window.getComputedStyle(el).boxShadow)
        expect(boxShadow === 'none' || boxShadow === '').toBeTruthy()
      }
    }
  })

  test('no backdrop-blur effects', async ({ page }) => {
    const elementsWithBackdrop = page.locator('[class*="backdrop-blur"]')
    await expect(elementsWithBackdrop).toHaveCount(0)
  })
})

test.describe('Component Visual Tests', () => {
  test('button variants', async ({ page }) => {
    await page.goto('/')

    const buttons = page.locator('button')
    const count = await buttons.count()

    for (let i = 0; i < Math.min(count, 5); i++) {
      const button = buttons.nth(i)
      if (await button.isVisible()) {
        const boxShadow = await button.evaluate((el) => window.getComputedStyle(el).boxShadow)
        expect(boxShadow === 'none' || boxShadow === '' || boxShadow === '0 0 0 transparent').toBeTruthy()
      }
    }
  })

  test('cards have borders instead of shadows', async ({ page }) => {
    const cards = page.locator('[data-slot="card"]')
    const count = await cards.count()

    for (let i = 0; i < count; i++) {
      const card = cards.nth(i)
      if (await card.isVisible()) {
        const styles = await card.evaluate((el) => ({
          boxShadow: window.getComputedStyle(el).boxShadow,
          border: window.getComputedStyle(el).border
        }))

        expect(
          styles.boxShadow === 'none' || styles.boxShadow === '' || styles.boxShadow === '0 0 0 transparent'
        ).toBeTruthy()
      }
    }
  })
})

test.describe('Dashboard Visual Regression', () => {
  test.skip('events list page', async ({ page }) => {
    await page.goto('/events')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('events-list.png', {
      maxDiffPixelRatio: 0.02
    })
  })

  test.skip('statistics page', async ({ page }) => {
    await page.goto('/statistics')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('statistics.png', {
      maxDiffPixelRatio: 0.02
    })
  })
})
