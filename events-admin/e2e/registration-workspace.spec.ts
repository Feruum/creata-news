import { expect, test } from '@playwright/test'

const organizerSession = {
  id: 'session-one',
  active: true,
  expires_at: '2099-01-01T00:00:00Z',
  identity: { id: 'organizer-one', traits: { email: 'organizer@example.test' } }
}

test.setTimeout(120_000)
test.use({ actionTimeout: 30_000 })

test('event details expose the registration workspace only to editors and registration managers', async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ status: 401, json: {} }))
  for (const permission of ['edit', 'manage', 'none']) {
    await page.goto(`/e2e/fixtures/registration-workspace.html?header&permission=${permission}`)
    await expect(page.getByRole('dialog', { name: 'Registration workspace event' })).toBeVisible()
    const entry = page.getByRole('link', { name: 'Форма и ответы', exact: true })
    if (permission === 'none') await expect(entry).toHaveCount(0)
    else {
      await expect(entry).toBeVisible()
      await expect(entry).toHaveAttribute('href', '/event-registration/42')
    }
  }
})

test('workspace requests wait for authentication and are cleared when the account expires', async ({ page }) => {
  const now = Date.now()
  await page.clock.install({ time: new Date(now) })
  let signedIn = false
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill(
      signedIn
        ? {
            json: {
              id: 'session-one',
              active: true,
              expires_at: new Date(now + 60_000).toISOString(),
              identity: { id: 'organizer-one', traits: { email: 'organizer@example.test' } }
            }
          }
        : { status: 401, json: {} }
    )
  )
  await page.goto('/e2e/fixtures/registration-workspace.html')
  await expect(page.getByRole('alert')).toContainText('войдите в аккаунт')
  await expect(page.locator('html')).not.toHaveAttribute('data-draft-requests')
  signedIn = true
  await page.getByRole('button', { name: 'Refresh account' }).click()
  await expect(page.getByTestId('account')).toHaveText('organizer-one')
  await expect(page.locator('html')).toHaveAttribute('data-draft-requests', '1')
  await page.getByRole('button', { name: 'Добавить вопрос', exact: true }).click()
  await page.getByRole('textbox', { name: 'Название вопроса', exact: true }).fill('Private unsaved question')
  signedIn = false
  await page.clock.fastForward(61_000)
  await expect(page.getByRole('alert')).toContainText(/Войдите|войдите/)
  await expect(page.locator('html')).toHaveAttribute('data-draft-requests', '1')
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true })).toHaveCount(0)
})

test('existing registration lists distinguish pending and rejected responses on both channels', async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ status: 401, json: {} }))
  await page.goto('/e2e/fixtures/registration-workspace.html?header&permission=manage&statuses')
  await expect(page.getByRole('cell', { name: 'Pending approval', exact: true })).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Rejected', exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'External visitors', exact: true }).click()
  await expect(page.getByRole('row').filter({ hasText: 'Pending visitor' })).toContainText('Pending approval')
  await expect(page.getByRole('row').filter({ hasText: 'Rejected visitor' })).toContainText('Rejected')
})

test('shared builder supports preview, moving questions, draft save and publication', async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ json: organizerSession }))
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/e2e/fixtures/registration-workspace.html')
  await page.getByRole('button', { name: 'Добавить вопрос', exact: true }).click()
  await page.getByRole('combobox', { name: 'Тип ответа', exact: true }).selectOption({ label: 'Один вариант' })
  await page.getByRole('textbox', { name: 'Название вопроса', exact: true }).fill('Планируете прийти?')
  await page.getByRole('textbox', { name: 'Вариант 1', exact: true }).fill('Да')
  await page.getByRole('button', { name: 'Добавить вариант', exact: true }).click()
  await page.getByRole('textbox', { name: 'Вариант 2', exact: true }).fill('Нет')
  await page.getByRole('button', { name: 'Добавить вопрос', exact: true }).click()
  await page.getByRole('textbox', { name: 'Название вопроса', exact: true }).fill('Почему?')
  await page.getByRole('checkbox', { name: 'Обязательный вопрос', exact: true }).check()
  await page.getByRole('button', { name: 'Переместить вопрос 2 вверх', exact: true }).click()
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true }).first()).toHaveValue('Почему?')
  await page.getByRole('button', { name: 'Переместить вопрос 1 вниз', exact: true }).click()
  await page.getByRole('button', { name: 'Предпросмотр', exact: true }).click()
  await expect(page.getByRole('textbox', { name: /Почему/ })).toBeVisible()
  await page.getByRole('radio', { name: 'Да', exact: true }).check()
  await page.getByRole('button', { name: 'Проверить анкету', exact: true }).click()
  await expect(page.getByRole('textbox', { name: /Почему/ })).toHaveAttribute('aria-invalid', 'true')
  await page.getByRole('textbox', { name: /Почему/ }).fill('Чтобы учиться')
  await page.getByRole('button', { name: 'Проверить анкету', exact: true }).click()
  await expect(page.getByText('Предпросмотр завершён. Ответы не отправлены.', { exact: true })).toBeVisible()
  await expect(page.locator('html')).not.toHaveAttribute('data-writes')
  await page.getByRole('button', { name: 'Редактирование', exact: true }).click()
  await page.getByRole('button', { name: 'Сохранить черновик', exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-writes', '["save"]')
  await page.getByRole('button', { name: 'Опубликовать', exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('data-writes', '["save","save","publish"]')
  await expect(page.getByText('Форма опубликована. Новые участники увидят эту версию.', { exact: true })).toBeVisible()
})

test('CRM review preserves pending details and unsaved notes on error, then confirms on explicit retry', async ({
  page
}) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ json: organizerSession }))
  await page.goto('/e2e/fixtures/registration-workspace.html?crm&reviewError')
  await page.getByRole('button', { name: 'Sunday participant', exact: true }).click()
  await expect(page.getByText('Ожидает решения', { exact: true })).toBeVisible()
  const note = page.getByRole('textbox', { name: 'Внутренняя заметка', exact: true })
  await note.fill('Contact participant before the workshop')
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('Не удалось сохранить решение')
  await expect(note).toHaveValue('Contact participant before the workshop')
  await expect(page.getByText('Ожидает решения', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Подтвердить', exact: true }).click()
  await expect(page.getByText('Подтверждено', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Подтвердить', exact: true })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Сохранить заметку', exact: true })).toHaveCount(0)
  await expect(page.getByText('Заметка сохранена', { exact: true })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('data-writes', /"note"/)
  await expect(page.locator('html')).toHaveAttribute('data-writes', /"review"/)
})

test('builder keeps edits across question disclosure and reveals a collapsed publication error', async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ json: organizerSession }))
  await page.goto('/e2e/fixtures/registration-workspace.html?seeded')
  const first = page.getByRole('button', { name: 'Вопрос 1: Почему вы хотите участвовать?', exact: true })
  const second = page.getByRole('button', { name: 'Вопрос 2: Какой у вас опыт?', exact: true })
  await expect(first).toHaveAttribute('aria-expanded', 'true')
  await expect(second).toHaveAttribute('aria-expanded', 'false')
  await page.getByRole('textbox', { name: 'Название вопроса', exact: true }).fill('')
  await second.click()
  await page.getByRole('textbox', { name: 'Вариант 2', exact: true }).fill('Практиковался самостоятельно')
  await page.getByRole('button', { name: 'Опубликовать', exact: true }).click()
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true })).toBeFocused()
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true })).toHaveAttribute(
    'aria-invalid',
    'true'
  )
  await expect(page.locator('html')).not.toHaveAttribute('data-writes')
  await page.getByRole('textbox', { name: 'Название вопроса', exact: true }).fill('Ваша мотивация')
  await second.click()
  await expect(page.getByRole('textbox', { name: 'Вариант 2', exact: true })).toHaveValue(
    'Практиковался самостоятельно'
  )
  await page.getByRole('button', { name: 'Предпросмотр', exact: true }).click()
  await expect(page.getByRole('textbox', { name: /Ваша мотивация/ })).toBeVisible()
  await expect(page.getByRole('radio', { name: 'Практиковался самостоятельно', exact: true })).toBeVisible()
})

test('registration answers stay read-only and attachment downloads remain available', async ({ page }) => {
  await page.route('**/sessions/whoami', (route) => route.fulfill({ json: organizerSession }))
  await page.goto('/e2e/fixtures/registration-workspace.html?seeded&crm')
  await page.getByRole('button', { name: 'Алия Садыкова', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Ответы на вопросы', exact: true })).toBeVisible()
  await expect(page.getByRole('link', { name: '@aliya_sadykova', exact: true })).toHaveAttribute(
    'href',
    'https://t.me/aliya_sadykova'
  )
  await expect(page.getByText('Хочу поработать над проектом с командой.', { exact: true })).toBeVisible()
  await expect(page.getByText('Есть небольшой опыт', { exact: true })).toBeVisible()
  await expect(page.getByText('portfolio.pdf', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Скачать', exact: true })).toBeEnabled()
  await expect(page.getByRole('textbox')).toHaveCount(1)
  await expect(page.getByRole('textbox', { name: 'Внутренняя заметка', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Подтвердить', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: 'Отклонить', exact: true }).click()
  await expect(page.getByText('Отклонено', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Вернуть на рассмотрение', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Подтвердить', exact: true })).toBeEnabled()
})

test('question motion can reverse and reduced motion keeps disclosure immediate', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'no-preference' })
  await page.goto('/e2e/fixtures/registration-workspace.html?design')
  const first = page.getByRole('button', { name: 'Вопрос 1: Почему вы хотите участвовать?', exact: true })
  const body = page.locator('.rf-field-collapse').first()
  await expect(first).toHaveAttribute('aria-expanded', 'true')
  const fullHeight = await body.evaluate((element) => element.getBoundingClientRect().height)
  await first.click()
  await expect(body).toHaveAttribute('inert', '')
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true })).toHaveCount(0)
  const intermediateHeight = await body.evaluate((element) => {
    const animations = element.getAnimations()
    const resize = animations.find(
      (animation) =>
        animation.effect instanceof KeyframeEffect &&
        animation.effect.getKeyframes().some((frame) => 'gridTemplateRows' in frame)
    )
    if (!resize) return null
    resize.pause()
    resize.currentTime = Number(resize.effect?.getComputedTiming().duration) / 2
    const height = element.getBoundingClientRect().height
    for (const animation of animations) animation.finish()
    return height
  })
  expect(intermediateHeight).not.toBeNull()
  expect(intermediateHeight).toBeGreaterThan(0)
  expect(intermediateHeight).toBeLessThan(fullHeight)
  await first.click()
  await first.click()
  await first.click()
  await expect(first).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByRole('textbox', { name: 'Название вопроса', exact: true })).toHaveValue(
    'Почему вы хотите участвовать?'
  )
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await expect(body).toHaveCSS('transition-duration', '0s')
  await first.click()
  await expect(first).toHaveAttribute('aria-expanded', 'false')
  expect(await body.evaluate((element) => element.getBoundingClientRect().height)).toBe(0)
  expect(await body.evaluate((element) => element.getAnimations().length)).toBe(0)
  const second = page.getByRole('button', { name: 'Вопрос 2: Какой у вас опыт?', exact: true })
  await second.focus()
  await page.keyboard.press('Space')
  await expect(second).toHaveAttribute('aria-expanded', 'true')
  await expect(page.getByRole('textbox', { name: 'Вариант 1', exact: true })).toHaveValue('Только начинаю')
})
