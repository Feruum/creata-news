import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      json: {
        id: 'audience-fixture-session',
        active: true,
        identity: { id: 'audience-fixture-user', traits: { email: 'staff@example.test' } }
      }
    })
  )
})

test('real audience statistics use completed accounts for every distribution', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-open-audience.html')
  const section = page.getByRole('region', { name: 'Аудитория Sunday Open' })
  await expect(section).toBeVisible()
  await expect(section.getByText('4 из 8 аккаунтов', { exact: true })).toBeVisible()
  const interests = section.getByRole('region', { name: 'Что интересно участникам' })
  await expect(interests.getByRole('meter', { name: 'IT и AI' })).toHaveAttribute('aria-valuenow', '75')
  await expect(interests.getByRole('meter', { name: 'Дизайн' })).toHaveAttribute('aria-valuenow', '75')
  await expect(section.getByRole('region', { name: 'Кто приходит' })).toContainText('Родитель')
  await expect(section.getByRole('region', { name: 'Зачем приходят' })).toContainText('Попробовать новое')
  await expect(section.getByRole('region', { name: 'Любимые форматы' })).toContainText('Воркшопы и мастер-классы')
  await expect(section.getByRole('region', { name: 'Как узнают о нас' })).toContainText('Instagram')
  await expect(section).not.toContainText('MOCK DATA')
})

test('empty audience has an explicit empty state without invalid percentages', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-open-audience.html?empty')
  await expect(page.getByText('Пока нет заполненных анкет', { exact: true })).toBeVisible()
  await expect(page.getByText('0 из 8 аккаунтов', { exact: true })).toBeVisible()
  await expect(page.getByRole('meter')).toHaveCount(0)
  await expect(page.locator('main')).not.toContainText('NaN')
})

test('audience errors offer retry and never substitute demo numbers', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-open-audience.html?error')
  await expect(page.getByRole('alert')).toContainText('Не удалось загрузить статистику')
  await expect(page.getByText('4 из 8 аккаунтов', { exact: true })).toHaveCount(0)
  await page.getByRole('button', { name: 'Повторить', exact: true }).click()
  await expect(page.getByText('4 из 8 аккаунтов', { exact: true })).toBeVisible()
})

test('non-staff accounts neither render audience statistics nor request its RPC', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-open-audience.html?denied')
  await expect(page.getByRole('region', { name: 'Аудитория Sunday Open' })).toHaveCount(0)
  await expect(page.getByRole('status', { name: 'Aggregate requests' })).toHaveText('0')
})

test('server permission failures do not disclose aggregate numbers', async ({ page }) => {
  await page.goto('/e2e/fixtures/sunday-open-audience.html?serverDenied')
  await expect(page.getByRole('alert')).toContainText('Нет доступа к статистике аудитории')
  await expect(page.getByText('4 из 8 аккаунтов', { exact: true })).toHaveCount(0)
})

test('audience statistics fit a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/e2e/fixtures/sunday-open-audience.html')
  await expect(page.getByText('4 из 8 аккаунтов', { exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
