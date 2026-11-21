import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({
    json: {
      id: 'external-visitors-fixture-session',
      active: true,
      identity: { id: 'external-visitors-fixture-user', traits: { email: 'account@example.test' } }
    }
  }))
})

test('account registrations without a phone display a safe placeholder', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html?missingPhone=true')
  await page.getByRole('tab', { name: 'External visitors' }).click()
  const row = page.getByRole('row').filter({ hasText: 'Event 1 visitor 1' }).first()
  await expect(row.getByRole('cell').nth(1)).toHaveText('—')
  await expect(row).not.toContainText('null')
  await expect(row).not.toContainText('undefined')
})

test('external visitors use independent server pages and preserve account participants', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html')
  await expect(page.getByText('@account-attendee', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await expect(page.getByText('Total: 26', { exact: true })).toBeVisible()
  await expect(page.getByText('Event 1 visitor 1', { exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Cancelled', exact: true })).toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'Phone', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Previous page', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: 'Next page', exact: true }).click()
  await expect(page.getByText('Event 1 visitor 26', { exact: true })).toBeVisible()
  await expect(page.getByText('Event 1 visitor 1', { exact: true })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Next page', exact: true })).toBeDisabled()
  await page.getByRole('tab', { name: 'EMS accounts', exact: true }).click()
  await expect(page.getByText('@account-attendee', { exact: true })).toBeVisible()
})

test('switching events resets the external visitors page', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html')
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await page.getByRole('button', { name: 'Next page', exact: true }).click()
  await expect(page.getByText('Event 1 visitor 26', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Next event', exact: true }).click()
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await expect(page.getByText('Event 2 visitor 1', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Previous page', exact: true })).toBeDisabled()
})

test('external visitor failures offer retry without displaying stale participant data', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html?error=true')
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await expect(page.getByRole('alert')).toContainText('Unable to load external visitors')
  await expect(page.getByText('Event 1 visitor 1', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Retry', exact: true }).click()
  await expect(page.getByText('Event 1 visitor 1', { exact: true })).toBeVisible()
})

test('empty external visitors show a separate zero total', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html?empty=true')
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await expect(page.getByText('No external visitors yet', { exact: true })).toBeVisible()
  await expect(page.getByText('Total: 0', { exact: true })).toBeVisible()
})

test('a denied external list displays the permission error', async ({ page }) => {
  await page.goto('/e2e/fixtures/external-registrations.html?denied=true')
  await page.getByRole('tab', { name: 'External visitors' }).click()
  await expect(page.getByRole('alert')).toContainText('You do not have permission to view external visitors')
})
