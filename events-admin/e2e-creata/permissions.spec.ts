import { expect, test, type Page } from '@playwright/test'

async function signIn(page: Page, role: string) {
  await page.goto('/today')
  await page.getByLabel('Роль', { exact: true }).selectOption(role)
  await page.getByRole('button', { name: 'Войти в локальной разработке', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Сегодня', exact: true })).toBeVisible()
}

const auditLink = (page: Page) => page.locator('[data-sidebar="sidebar"] a[href="/audit"]')

test('authors read digests and catalogs without controls or requests for restricted actions', async ({ page }) => {
  const restrictedReads: string[] = []
  page.on('request', request => {
    const path = new URL(request.url()).pathname
    if (path.endsWith('/admin/audit') || path.endsWith('/admin/deliveries')) restrictedReads.push(path)
  })
  await signIn(page, 'author')

  await expect(auditLink(page)).toHaveCount(0)
  await page.goto('/audit')
  await expect(page.getByText('У вашей роли нет доступа к этому действию.', { exact: true })).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)

  const digestRead = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/admin/digests'))
  await page.goto('/digests')
  await expect(page.getByRole('heading', { name: 'Предпросмотр', exact: true })).toBeVisible()
  expect((await digestRead).ok()).toBe(true)
  await expect(page.getByRole('button', { name: 'Собрать выпуск', exact: true })).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Доставка выпусков', exact: true })).toHaveCount(0)
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(restrictedReads).toEqual([])

  await page.goto('/authors')
  await expect(page.locator('.author-card').first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Добавить автора', exact: true })).toHaveCount(0)
  await expect(page.locator('.author-card button')).toHaveCount(0)

  await page.goto('/opportunities')
  const row = page.locator('tbody tr').first()
  await expect(row).toBeVisible()
  await expect(row.getByRole('link')).toBeVisible()
  await expect(row.getByRole('button')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Добавить возможность', exact: true })).toHaveCount(0)
})

test('editors manage digests and catalogs while the audit remains restricted', async ({ page }) => {
  await signIn(page, 'editor')
  await expect(auditLink(page)).toHaveCount(0)

  await page.goto('/digests')
  await expect(page.getByRole('button', { name: 'Собрать выпуск', exact: true })).toBeEnabled()
  await expect(page.getByRole('heading', { name: 'Доставка выпусков', exact: true })).toBeVisible()

  await page.goto('/authors')
  await expect(page.getByRole('button', { name: 'Добавить автора', exact: true })).toBeEnabled()
  await expect(page.locator('.author-card button').first()).toBeEnabled()

  await page.goto('/opportunities')
  await expect(page.getByRole('button', { name: 'Добавить возможность', exact: true })).toBeEnabled()
  await expect(page.locator('tbody tr').first().getByRole('button')).toBeEnabled()
})

test('the chief editor can reach the audit from the sidebar', async ({ page }) => {
  const auditRead = page.waitForResponse(response =>
    new URL(response.url()).pathname.endsWith('/admin/audit') && response.request().method() === 'GET',
  )
  await signIn(page, 'chief_editor')
  await auditLink(page).click()
  await expect(page.getByRole('heading', { name: 'Журнал', exact: true })).toBeVisible()
  expect((await auditRead).ok()).toBe(true)
})
