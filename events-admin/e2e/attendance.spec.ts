import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.clock.install({ time: new Date('2026-10-11T12:00:00Z') })
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      headers: {
        'access-control-allow-origin': new URL(page.url()).origin,
        'access-control-allow-credentials': 'true'
      },
      json: {
        id: 'attendance-session',
        active: true,
        expires_at: '2026-10-12T12:00:00Z',
        identity: { id: 'attendance-user', traits: { email: 'attendee@test.local' } }
      }
    })
  )
})

test('an open event updates attendance after a check-in from another client', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html')
  await expect(page.getByText('Checked In: 0', { exact: true })).toBeVisible()
  const metadata = page.getByText('Account Attendees', { exact: true }).locator('..')
  await expect(metadata).toContainText('0')

  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.clock.fastForward(6000)
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
  await expect(metadata).toContainText('1')
  await expect(page.getByText('Attended: 0', { exact: true })).toBeVisible()
  await expect
    .poll(async () => JSON.parse((await page.getByTestId('requests').textContent()) || '{}').registrations)
    .toBeGreaterThan(1)

  await page.getByRole('button', { name: 'Close', exact: true }).click()
  const closed = await page.getByTestId('requests').textContent()
  await page.clock.fastForward(20000)
  await expect(page.getByTestId('requests')).toHaveText(closed || '')
})

test('attendance badges use full server totals instead of the returned rows', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html?many=true')
  await expect(page.getByText('Registered: 500', { exact: true })).toBeVisible()
  await expect(page.getByText('Checked In: 100', { exact: true })).toBeVisible()
  await expect(page.getByText('Attended: 1', { exact: true })).toBeVisible()
  await expect(page.getByText('No Show: 3', { exact: true })).toBeVisible()
  await expect(page.getByText('Account Attendees', { exact: true }).locator('..')).toContainText('101')
})

test('reopening a locked event refreshes cached attendance without polling', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html?locked=true')
  await expect(page.getByText('Checked In: 0', { exact: true })).toBeVisible()
  const initial = await page.getByTestId('requests').textContent()
  await page.clock.fastForward(6000)
  await expect(page.getByTestId('requests')).toHaveText(initial || '')
  await page.getByRole('button', { name: 'Close', exact: true }).click()
  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.getByRole('button', { name: 'Open event', exact: true }).click()
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
  await expect(page.getByText('Account Attendees', { exact: true }).locator('..')).toContainText('1')
})

test('immediately reopening a locked event refreshes even a fresh suspense cache', async ({ page }) => {
  await page.clock.setFixedTime(new Date('2026-10-11T12:00:00Z'))
  await page.goto('/e2e/fixtures/attendance.html?locked=true')
  const metadata = page.getByText('Account Attendees', { exact: true }).locator('..')
  await expect(metadata).toContainText('0')
  await page.getByRole('button', { name: 'Close', exact: true }).click()
  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.getByRole('button', { name: 'Open event', exact: true }).click()
  await expect(metadata).toContainText('1')
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
})

test('a locked event refreshes on focus even while its suspense cache is fresh', async ({ page }) => {
  await page.clock.setFixedTime(new Date('2026-10-11T12:00:00Z'))
  await page.goto('/e2e/fixtures/attendance.html?locked=true')
  const metadata = page.getByText('Account Attendees', { exact: true }).locator('..')
  await expect(metadata).toContainText('0')
  await page.evaluate(() => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
    window.dispatchEvent(new Event('external-check-in'))
  })
  await page.evaluate(() => {
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
  })
  await expect(metadata).toContainText('1')
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
})

test('attendance conversion updates on the open analytics screen', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html?analytics=true')
  await expect(page.getByText('0.0%', { exact: true })).toBeVisible()
  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.clock.fastForward(16000)
  await expect(page.getByText('100.0%', { exact: true })).toBeVisible()
  await expect(page.getByText('1 attended', { exact: true })).toBeVisible()
})

for (const analytics of [false, true]) {
  test(`${analytics ? 'analytics' : 'event details'} pause polling in the background and refresh on focus`, async ({
    page
  }) => {
    await page.goto(`/e2e/fixtures/attendance.html${analytics ? '?analytics=true' : ''}`)
    await expect(page.getByText(analytics ? '0.0%' : 'Checked In: 0', { exact: true })).toBeVisible()
    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
      document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
      window.dispatchEvent(new Event('external-check-in'))
    })
    const hidden = await page.getByTestId('requests').textContent()
    await page.clock.fastForward(20000)
    await expect(page.getByTestId('requests')).toHaveText(hidden || '')

    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
      document.dispatchEvent(new Event('visibilitychange', { bubbles: true }))
    })
    await expect(page.getByText(analytics ? '100.0%' : 'Checked In: 1', { exact: true })).toBeVisible()
  })
}

test('grace period attendance updates at the live polling interval', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html?grace=true')
  await expect(page.getByText('Checked In: 0', { exact: true })).toBeVisible()
  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.clock.fastForward(6000)
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
})

test('an event opened before check-in activates begins polling when it becomes live', async ({ page }) => {
  await page.goto('/e2e/fixtures/attendance.html?upcoming=true')
  await expect(page.getByText('Checked In: 0', { exact: true })).toBeVisible()
  await expect(page.getByText('Check-in Phase', { exact: true }).locator('..')).toContainText('Not started')
  await page.clock.fastForward(11000)
  await expect(page.getByText('Check-in Phase', { exact: true }).locator('..')).toContainText('Live')
  await page.evaluate(() => window.dispatchEvent(new Event('external-check-in')))
  await page.clock.fastForward(6000)
  await expect(page.getByText('Checked In: 1', { exact: true })).toBeVisible()
  await expect(page.getByText('Account Attendees', { exact: true }).locator('..')).toContainText('1')
})
