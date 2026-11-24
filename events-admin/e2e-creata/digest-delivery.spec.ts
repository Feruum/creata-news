import { expect, test, type Page } from '@playwright/test'

const edition = {
  id: 'test-edition',
  kind: 'channel',
  locale: 'ru',
  status: 'approved',
  articles: [],
  opportunities: [],
  createdAt: '2026-11-24T04:00:00Z',
  text: 'Тестовый выпуск',
}

async function openApprovedEdition(page: Page, channelDeliveryStatus: 'available' | 'channel_missing') {
  await page.route('**/api/creata/v1/admin/digests', route =>
    route.fulfill({ json: { items: [edition], channelDeliveryStatus } }),
  )
  await page.goto('/digests?edition=test-edition')
  await page.getByLabel('Роль', { exact: true }).selectOption('chief_editor')
  await page.getByRole('button', { name: 'Войти в локальной разработке', exact: true }).click()
  await expect(page.getByText('Тестовый выпуск', { exact: true })).toBeVisible()
}

test('the approved edition explains why an unconfigured channel cannot send', async ({ page }) => {
  await openApprovedEdition(page, 'channel_missing')
  await expect(page.getByRole('button', { name: 'Отправить в канал', exact: true })).toBeDisabled()
  await expect(page.getByText('Отправка недоступна. Укажите ID Telegram-канала в настройках сервера.', { exact: true })).toBeVisible()
})

test('a rejected send is visible inside the confirmation dialog', async ({ page }) => {
  await page.route('**/api/creata/v1/admin/digests/test-edition/send', route =>
    route.fulfill({ status: 503, json: { error: 'Test delivery failure' } }),
  )
  await openApprovedEdition(page, 'available')
  await page.getByRole('button', { name: 'Отправить в канал', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: 'Отправить в канал' })
  await dialog.getByRole('button', { name: 'Отправить в канал', exact: true }).click()
  await expect(dialog.getByRole('alert')).toBeVisible()
})
