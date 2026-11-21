import { expect, test } from '@playwright/test'

test('creating Music makes its actual server ID available in the event filter', async ({ page }) => {
  await page.goto('/e2e/fixtures/category-suggestions.html')
  await page.getByRole('button', { name: 'Add Music category', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Music category already exists', exact: true })).toBeDisabled()
  const filter = page.getByRole('combobox', { name: 'Category filter' })
  await filter.selectOption({ label: 'Music' })
  await expect(filter).toHaveValue('71')
  await expect(page.getByTestId('category-requests')).toHaveText(JSON.stringify([{ name: 'Music', color: '#8B5CF6' }]))
})

test('existing names cannot be added again even with different case and spaces', async ({ page }) => {
  await page.goto('/e2e/fixtures/category-suggestions.html?existingMusic=true')
  await expect(page.getByRole('button', { name: 'Music category already exists', exact: true })).toBeDisabled()
  await expect(page.getByTestId('category-requests')).toHaveText('[]')
  await page.getByRole('button', { name: 'Add Design category', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Design category already exists', exact: true })).toBeDisabled()
})

test('failed category creation stays retryable and does not add a false filter option', async ({ page }) => {
  await page.goto('/e2e/fixtures/category-suggestions.html?failFirst=true')
  await page.getByRole('button', { name: 'Add Music category', exact: true }).click()
  await expect(page.getByText('Could not create Music. Please try again.', { exact: true })).toBeVisible()
  await expect(page.getByRole('option', { name: 'Music', exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Add Music category', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Music category already exists', exact: true })).toBeDisabled()
  await expect(page.getByRole('option', { name: 'Music', exact: true })).toHaveCount(1)
})

test('suggestions fit a mobile viewport and can be created with the keyboard', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/e2e/fixtures/category-suggestions.html')
  const music = page.getByRole('button', { name: 'Add Music category', exact: true })
  await expect(music).toBeEnabled()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await music.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: 'Music category already exists', exact: true })).toBeDisabled()
})
