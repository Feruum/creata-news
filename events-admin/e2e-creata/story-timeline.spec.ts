import { expect, test, type Page } from '@playwright/test'

async function editorialAction(page: Page, name: string) {
  await page.locator('.editor-controls').getByRole('button', { name, exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByRole('button', { name, exact: true }).click()
  await expect(dialog).toBeHidden()
}

async function publish(page: Page) {
  await editorialAction(page, 'На проверку')
  await editorialAction(page, 'Одобрить')
  await editorialAction(page, 'Опубликовать')
  await expect(page.locator('.page-actions .badge')).toHaveText('Опубликовано')
}

async function saveDraft(page: Page) {
  const response = page.waitForResponse(result =>
    result.request().method() === 'PATCH' && new URL(result.url()).pathname.startsWith('/api/creata/v1/admin/articles/'),
  )
  const button = page.locator('.editor-controls').getByRole('button', { name: 'Сохранить', exact: true })
  await button.click()
  const saved = await response
  expect(saved.ok(), await saved.text()).toBe(true)
  await expect(button).toBeDisabled()
}

async function addUpdate(page: Page, number: number, date: string, title: string, body: string, link: string) {
  await page.getByRole('button', { name: 'Добавить обновление', exact: true }).click()
  const update = page.getByRole('group', { name: `Обновление ${number}`, exact: true })
  await update.getByLabel('Дата и время обновления (UTC)', { exact: true }).fill(date)
  await update.getByLabel('Заголовок обновления', { exact: true }).fill(title)
  await update.getByLabel('Текст обновления', { exact: true }).fill(body)
  await update.getByLabel('Источник обновления', { exact: true }).fill(link)
  return update
}

test('editor publishes a developing story and a reader follows its next approved update', async ({ page, browser }) => {
  test.setTimeout(120_000)
  const web = process.env.CREATA_WEB_TEST_URL || 'http://localhost:3002'
  const marker = `Акселератор UI ${Date.now()}`
  await page.goto('/')
  await page.getByLabel('Роль', { exact: true }).selectOption('chief_editor')
  await page.getByRole('button', { name: 'Войти в локальной разработке', exact: true }).click()
  await page.locator('.page-actions').getByRole('button', { name: 'Новый материал', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('Заголовок', { exact: true }).fill(marker)
  await dialog.getByLabel('Рубрика', { exact: true }).selectOption('ai')
  await dialog.getByRole('button', { name: 'Новый материал', exact: true }).click()
  await expect(page).toHaveURL(/\/articles\/[a-z0-9-]+$/)

  const editor = page.locator('.editor-main')
  for (const language of ['RU', 'ҚАЗ', 'EN']) {
    await editor.getByRole('button', { name: language, exact: true }).click()
    await editor.getByLabel('Заголовок', { exact: true }).fill(language === 'RU' ? marker : `${marker} ${language}`)
    await editor.getByLabel('Лид', { exact: true }).fill(`Появилась новая программа • ${language}`)
    await editor.getByLabel('Текст статьи', { exact: true }).fill(`Редакционная история о программе • ${language}`)
    await editor.getByLabel('Краткое содержание', { exact: true }).fill(`Этапы программы • ${language}`)
  }
  await editor.getByRole('button', { name: 'RU', exact: true }).click()
  await page.getByRole('button', { name: 'Развитие сюжета', exact: true }).click()
  await page.getByRole('button', { name: 'Добавить инструкцию', exact: true }).click()
  await page.getByLabel('Для кого', { exact: true }).fill('Команды с прототипом')
  await page.getByLabel('Что делать', { exact: true }).fill('Подготовьте демо и откройте форму организатора.')
  await page.getByLabel('Источник инструкции', { exact: true }).fill('https://example.org/creata-accelerator/rules')
  const announcement = await addUpdate(page, 1, '2026-11-15T09:00', 'Анонс программы', 'Организатор представил программу.', 'https://example.org/creata-accelerator/announcement')
  await announcement.getByLabel('Слаг связанного материала', { exact: true }).fill('akku-vtoraya-zhizn')
  await editor.getByRole('button', { name: 'EN', exact: true }).click()
  await page.getByLabel('Для кого', { exact: true }).fill('Teams with a prototype')
  await page.getByLabel('Что делать', { exact: true }).fill('Prepare a demo and open the application form.')
  await announcement.getByLabel('Заголовок обновления', { exact: true }).fill('Program announced')
  await announcement.getByLabel('Текст обновления', { exact: true }).fill('The organizer announced the program.')
  await editor.getByRole('button', { name: 'RU', exact: true }).click()
  await editor.getByRole('button', { name: 'Живой предпросмотр', exact: true }).click()
  const preview = editor.getByRole('region', { name: 'Живой предпросмотр' })
  await expect(preview.getByRole('heading', { name: 'Анонс программы', exact: true })).toBeVisible()
  await expect(preview.getByRole('link', { name: 'Источник инструкции', exact: true })).toHaveAttribute('href', 'https://example.org/creata-accelerator/rules')
  await saveDraft(page)

  const articleId = new URL(page.url()).pathname.split('/').pop()
  const draft = await page.request.get(`/api/creata/v1/admin/articles/${articleId}`)
  await expect(draft).toBeOK()
  const { slug } = await draft.json()
  const articlePath = `/ru/articles/${encodeURIComponent(slug)}`
  const articleUrl = new URL(articlePath, web).href
  expect((await page.request.get(`/api/creata/v1/public/articles/${encodeURIComponent(slug)}`)).status()).toBe(404)

  const readerContext = await browser.newContext({ viewport: { width: 390, height: 850 } })
  const reader = await readerContext.newPage()
  const followedArticle = reader.getByRole('link', { name: marker, exact: true }).and(reader.locator('a[href*="#story-update-"]'))
  const errors: string[] = []
  reader.on('pageerror', error => errors.push(error.message))
  try {
    await publish(page)
    await expect(page.getByLabel('Адрес статьи', { exact: true })).toBeDisabled()
    await reader.goto(new URL(`/en/articles/${encodeURIComponent(slug)}`, web).href)
    await expect(reader.getByRole('region', { name: 'Story timeline' })).toContainText('Program announced')
    await expect(reader.getByRole('region', { name: 'What to do next' })).toContainText('Teams with a prototype')
    await reader.goto(articleUrl)
    const timeline = reader.getByRole('region', { name: 'Хронология сюжета' })
    await expect(timeline.getByRole('heading', { name: 'Анонс программы' })).toBeVisible()
    await expect(timeline.getByRole('link', { name: 'Связанный материал' })).toHaveAttribute('href', '/ru/articles/akku-vtoraya-zhizn')
    const guide = reader.getByRole('region', { name: 'Практические шаги' })
    await expect(guide).toContainText('Команды с прототипом')
    await expect(guide).toContainText('Подготовьте демо')
    await expect(guide.getByRole('link', { name: 'Первоисточник' })).toHaveAttribute('href', 'https://example.org/creata-accelerator/rules')
    expect(await reader.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)

    await timeline.getByRole('link', { name: 'Следить за сюжетом' }).click()
    await reader.getByRole('button', { name: 'Local development sign-in' }).click()
    await expect(reader).toHaveURL(articleUrl)
    await timeline.getByRole('button', { name: 'Следить за сюжетом', exact: true }).click()
    await expect(timeline.getByRole('button', { name: 'Отписаться от сюжета', exact: true })).toHaveAttribute('aria-pressed', 'true')
    await timeline.getByRole('link', { name: 'Мои сюжеты' }).click()
    await expect(followedArticle).toBeVisible()
    await expect(reader.getByText('Анонс программы', { exact: true })).toBeVisible()

    await page.getByRole('button', { name: 'Развитие сюжета', exact: true }).click()
    await addUpdate(page, 2, '2026-11-18T10:30', 'Объявлены результаты', 'Организатор назвал отобранные команды.', 'https://example.org/creata-accelerator/results')
    await saveDraft(page)
    await page.getByRole('button', { name: 'История изменений', exact: true }).click()
    const revisions = page.getByRole('dialog', { name: 'Сравнить редакции' })
    await revisions.getByRole('combobox', { name: 'Раздел для сравнения' }).selectOption({ label: 'Хронология' })
    await expect(revisions.locator('.diff-line').filter({ hasText: 'Объявлены результаты' })).toBeVisible()
    await revisions.getByRole('button', { name: 'Закрыть', exact: true }).click()
    await reader.reload()
    await expect(reader.getByText('Анонс программы', { exact: true })).toBeVisible()
    await expect(reader.getByText('Объявлены результаты', { exact: true })).toHaveCount(0)

    await publish(page)
    await reader.reload()
    await expect(reader.getByText('Объявлены результаты', { exact: true })).toBeVisible()
    await followedArticle.click()
    await expect(reader).toHaveURL(new RegExp('/articles/[^#]+#story-update-'))
    expect(decodeURIComponent(new URL(reader.url()).pathname)).toBe(decodeURIComponent(articlePath))
    await expect(timeline.getByRole('heading', { level: 3 }).first()).toHaveText('Объявлены результаты')
    await expect(timeline.getByRole('heading', { level: 3 }).last()).toHaveText('Анонс программы')
    await expect(timeline.locator('time').first()).toHaveAttribute('datetime', /^2026-11-18T10:30:00(?:\.000)?Z$/)
    await timeline.getByRole('button', { name: 'Отписаться от сюжета', exact: true }).click()
    await expect(timeline.getByRole('button', { name: 'Следить за сюжетом', exact: true })).toHaveAttribute('aria-pressed', 'false')
    await reader.goto(new URL('/ru/following', web).href)
    await expect(reader.getByText('Вы пока не следите за сюжетами')).toBeVisible()
    expect(errors).toEqual([])
  } finally {
    await readerContext.close()
    if (await page.locator('.editor-controls').getByRole('button', { name: 'Снять с публикации', exact: true }).isEnabled().catch(() => false)) {
      await editorialAction(page, 'Снять с публикации')
    }
  }
})
