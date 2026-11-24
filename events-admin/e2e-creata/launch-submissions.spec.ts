import { expect, test } from '@playwright/test'

test('a founder revises a launch and an editor publishes it from the review queue', async ({ browser, page }) => {
  const web = process.env.CREATA_WEB_TEST_URL || 'http://localhost:3002'
  const name = `Launch review ${Date.now()}`
  const feedback = 'Добавьте, кто отвечает за разработку.'
  const team = 'Two founders and a lead developer'
  const founder = await browser.newContext({ viewport: { width: 390, height: 844 } })
  const publicPage = await founder.newPage()
  page.setDefaultTimeout(15_000)
  publicPage.setDefaultTimeout(15_000)
  try {
    await publicPage.goto(`${web}/ru/launches`)
    await publicPage.getByRole('link', { name: 'Подать запуск' }).click()
    await publicPage.getByRole('button', { name: 'Local development sign-in' }).click()
    const submissions = publicPage.getByRole('region', { name: 'Мои заявки на запуск' })
    await submissions.getByRole('button', { name: 'Подать запуск' }).click()
    await submissions.getByLabel('Название продукта').fill(name)
    await submissions.getByLabel('Ссылка на продукт').fill('https://example.org/startup')
    await submissions.getByLabel('Дата запуска').fill('2026-11-23')
    await submissions.getByRole('combobox', { name: 'Язык', exact: true }).selectOption('en')
    await submissions.getByLabel('Кратко о продукте').fill('A tool for small teams')
    await submissions.getByLabel('Описание продукта').fill('A place for small teams to plan their product releases.')
    await submissions.getByRole('textbox', { name: 'Команда', exact: true }).fill('Two founders')
    await submissions.getByRole('button', { name: 'Сохранить черновик' }).click()
    const submission = submissions.getByRole('article').filter({ hasText: name })
    await expect(submission).toContainText('Черновик')
    await submission.getByRole('button', { name: 'Отправить на модерацию' }).click()
    await expect(submission).toContainText('На модерации')
    expect(await publicPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)

    const before = await publicPage.request.get(`${web}/api/creata/v1/public/launches?q=${encodeURIComponent(name)}`)
    expect((await before.json()).items).toEqual([])
    await page.goto('/launches')
    await page.getByLabel('Роль', { exact: true }).selectOption('editor')
    await page.getByRole('button', { name: 'Войти в локальной разработке', exact: true }).click()
    const reviewButton = page.getByRole('button', { name: `Проверить заявку · ${name}` })
    await reviewButton.click()
    let dialog = page.getByRole('dialog', { name })
    await expect(dialog).toContainText('Two founders')
    await expect(dialog.getByRole('button', { name: 'Вернуть на доработку' })).toBeDisabled()
    await dialog.getByLabel('Комментарий для основателя').fill(feedback)
    await dialog.getByRole('button', { name: 'Вернуть на доработку' }).click()
    await expect(reviewButton).toHaveCount(0)

    await publicPage.reload()
    await expect(submission).toContainText(feedback)
    await submission.getByRole('button', { name: 'Редактировать' }).click()
    await submissions.getByRole('combobox', { name: 'Язык', exact: true }).selectOption('en')
    await submissions.getByRole('textbox', { name: 'Команда', exact: true }).fill(team)
    await submissions.getByRole('button', { name: 'Сохранить черновик' }).click()
    await submission.getByRole('button', { name: 'Отправить на модерацию' }).click()
    await expect(submission).toContainText('На модерации')
    await page.reload()
    await reviewButton.click()
    dialog = page.getByRole('dialog', { name })
    await expect(dialog).toContainText(team)
    await dialog.getByRole('button', { name: 'Опубликовать', exact: true }).click()
    await expect(reviewButton).toHaveCount(0)

    await publicPage.reload()
    await expect(submission).toContainText('Опубликовано')
    await submission.getByRole('link', { name: 'Открыть публикацию' }).click()
    await expect(publicPage.getByRole('heading', { level: 1, name })).toBeVisible()
    await expect(publicPage.getByText(team, { exact: true })).toHaveAttribute('lang', 'en')
  } finally {
    await founder.close()
  }
})
