import { test, expect } from '@playwright/test'
import { resolve } from 'node:path'

test('editor creates, edits, approves and publishes the same revision through the UI', async ({ page }) => {
  const errors: string[] = []; page.on('pageerror', e => errors.push(e.message))
  await page.goto('/')
  await page.getByLabel('Роль', {exact:true}).selectOption('chief_editor')
  await page.getByRole('button', {name:'Войти в локальной разработке',exact:true}).click()
  await page.getByRole('button', {name:'Новый материал',exact:true}).click()
  const modal = page.getByRole('dialog')
  const marker = `UI editorial ${Date.now()}`
  await modal.getByLabel('Заголовок', {exact:true}).fill(marker)
  await modal.getByLabel('Рубрика', {exact:true}).selectOption('ai')
  await modal.getByRole('button', {name:'Новый материал',exact:true}).click()
  await expect(page).toHaveURL(/\/articles\/[a-z0-9-]+$/)
  const editor = page.locator('.editor-main')
  for (const lang of ['RU','ҚАЗ','EN']) {
    await editor.getByRole('button', {name:lang,exact:true}).click()
    await editor.getByLabel('Заголовок', {exact:true}).fill(`${marker} ${lang}`)
    await editor.getByLabel('Текст статьи', {exact:true}).fill(`A complete source-backed story for ${lang}.\nThe editor can revise this text without losing other languages.`)
    await editor.getByLabel('Лид', {exact:true}).fill(`Lead ${lang}`)
    await editor.getByLabel('Краткое содержание', {exact:true}).fill(`Summary ${lang}`)
  }
  await page.getByRole('button', {name:'Сохранить',exact:true}).click()
  await expect(page.getByRole('button', {name:'Одобрить',exact:true})).toBeEnabled()
  await page.getByRole('button', {name:'Одобрить',exact:true}).click()
  await page.getByRole('dialog').getByRole('button', {name:'Одобрить',exact:true}).click()
  await expect(page.getByRole('button', {name:'Опубликовать',exact:true})).toBeEnabled()
  await page.getByRole('button', {name:'Опубликовать',exact:true}).click()
  if (process.env.CREATA_TEST_POSTIZ_CHANNEL) {
    await modal.getByRole('checkbox', {name:'Creata QA (threads)',exact:true}).check()
    await modal.getByRole('combobox', {name:'Язык поста · Creata QA'}).selectOption('en')
    await expect(modal.locator('.callout').filter({hasText:marker + ' EN'})).toContainText('/en/articles/')
  }
  await page.getByRole('dialog').getByLabel('Комментарий редактора', {exact:true}).fill('Checked in the disposable development database.')
  await page.getByRole('dialog').getByRole('button', {name:'Опубликовать',exact:true}).click()
  await expect(page.locator('.page-actions .badge')).toHaveText('Опубликовано')
  if (process.env.CREATA_TEST_POSTIZ_CHANNEL) {
    const deliveries = page.getByRole('region', {name:'Публикации в соцсетях'})
    await expect(deliveries).toContainText('Черновик в Postiz', {timeout:30_000})
    await expect(deliveries.getByRole('listitem')).toHaveCount(1)
  }
  await page.locator('.admin-scroll').evaluate(element=>element.scrollTo(0,0))
  await page.screenshot({path:resolve(import.meta.dirname,'../../.runtime/screenshots/editor-ui-workflow.png'),fullPage:true})
  await page.getByRole('button', {name:'Снять с публикации',exact:true}).click()
  await page.getByRole('dialog').getByRole('button', {name:'Снять с публикации',exact:true}).click()
  await expect(page.locator('.page-actions .badge')).toHaveText('На проверке')
  await page.goto('/today')
  await expect(page.getByRole('heading',{name:'Сегодня',exact:true})).toBeVisible()
  for (const width of [1440,768,360]) {
    await page.setViewportSize({width,height:1000})
    await expect(page.locator('.kanban')).toBeVisible()
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true)
    await page.screenshot({path:resolve(import.meta.dirname,`../../.runtime/screenshots/editorial-original-owner-${width}.png`)})
  }
  expect(errors).toEqual([])
})

test('Hermes source form persists an inactive source without starting collection', async ({ page }) => {
  const errors: string[] = [];page.on('pageerror', e => errors.push(e.message))
  await page.goto(process.env.CREATA_HERMES_TEST_URL || 'http://localhost:5174')
  await page.getByLabel('Роль', {exact:true}).selectOption('admin')
  await page.getByRole('button', {name:'Войти в локальной разработке',exact:true}).click()
  await page.locator('.page-actions').getByRole('button', {name:'Добавить источник',exact:true}).click()
  const modal=page.getByRole('dialog'),name=`UI RSS ${Date.now()}`
  await modal.getByLabel('Имя', {exact:true}).fill(name)
  await modal.getByLabel('Ссылка', {exact:true}).fill('https://example.com/creata-test-feed')
  await modal.getByLabel('Интервал сбора, мин', {exact:true}).fill('15')
  await modal.getByLabel('Включён', {exact:true}).uncheck()
  await modal.getByRole('button', {name:'Сохранить',exact:true}).click()
  await expect(modal).not.toBeVisible()
  const card=page.locator('.source-card').filter({has:page.getByRole('heading',{name,exact:true})})
  await expect(card).toBeVisible()
  await expect(card.getByRole('button',{name:'Выключен',exact:true})).toHaveAttribute('aria-pressed','false')
  await page.screenshot({path:resolve(import.meta.dirname,'../../.runtime/screenshots/hermes-original-owner-desktop.png')})
  expect(errors).toEqual([])
})

test('opportunity form preserves the photo and audience translations when editing', async ({page})=>{
  await page.goto('/opportunities')
  await page.getByLabel('Роль',{exact:true}).selectOption('chief_editor')
  await page.getByRole('button',{name:'Войти в локальной разработке',exact:true}).click()
  await page.getByRole('button',{name:'Добавить возможность',exact:true}).click()
  const modal=page.getByRole('dialog'),marker=`UI opportunity ${Date.now()}`
  for (const lang of ['RU','ҚАЗ','EN']) {
    await modal.getByRole('button',{name:lang,exact:true}).click()
    await modal.getByLabel('Заголовок',{exact:true}).fill(`${marker} ${lang}`)
    await modal.getByLabel('Для кого',{exact:true}).fill(`Audience ${lang}`)
  }
  await modal.getByRole('button',{name:'RU',exact:true}).click()
  await modal.getByLabel('Организация',{exact:true}).fill('Creata UI test')
  await modal.getByLabel('Дедлайн',{exact:true}).fill('2027-12-31T09:00')
  await modal.getByLabel('Ссылка',{exact:true}).fill('https://example.com/opportunity')
  await modal.getByLabel('Ссылка на фото',{exact:true}).fill('https://example.com/opportunity.jpg')
  await modal.getByRole('button',{name:'Сохранить',exact:true}).click()
  await expect(modal).not.toBeVisible()
  await page.getByRole('button',{name:`${marker} RU`,exact:true}).click()
  await expect(modal.getByLabel('Ссылка на фото',{exact:true})).toHaveValue('https://example.com/opportunity.jpg')
  await modal.getByLabel('Организация',{exact:true}).fill('Updated Creata UI test')
  await modal.getByRole('button',{name:'Сохранить',exact:true}).click()
  await expect(modal).not.toBeVisible()
  await page.getByRole('button',{name:`${marker} RU`,exact:true}).click()
  for (const lang of ['RU','ҚАЗ','EN']) {
    await modal.getByRole('button',{name:lang,exact:true}).click()
    await expect(modal.getByLabel('Для кого',{exact:true})).toHaveValue(`Audience ${lang}`)
  }
  await modal.getByLabel('Статус',{exact:true}).selectOption('rejected')
  await modal.getByRole('button',{name:'Сохранить',exact:true}).click()
  await expect(modal).not.toBeVisible()
})
