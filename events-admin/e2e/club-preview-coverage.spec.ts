import { expect, type Page, test } from '@playwright/test'

async function updates(page: Page) {
  const log: { method: string; input: Record<string, unknown> }[] = JSON.parse(
    (await page.getByTestId('requests').textContent()) || '[]'
  )
  return log.filter((entry) => entry.method === 'UpdateOrganization').map((entry) => entry.input)
}

test('all organization fields save together and every optional link and description can be cleared', async ({
  page
}) => {
  await page.setViewportSize({ width: 1440, height: 1080 })
  await page.goto('/admin')
  const preview = page.getByTestId('club-preview')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await preview.getByLabel('Name', { exact: true }).fill('Complete club edit')
  const description = preview.locator('[contenteditable="true"]')
  await description.click()
  await description.press('ControlOrMeta+A')
  await page.keyboard.insertText('Updated description from the club editor')
  await preview.getByRole('combobox', { name: 'Type', exact: true }).click()
  await page.getByRole('option', { name: 'Community', exact: true }).click()
  await preview.getByRole('combobox', { name: 'Status', exact: true }).click()
  await page.getByRole('option', { name: 'Frozen', exact: true }).click()
  const links = [
    ['Website', 'website', 'https://club.example.test'],
    ['Instagram', 'instagram', 'https://instagram.com/completeclub'],
    ['Telegram Channel', 'telegramChannel', 'https://t.me/completeclub'],
    ['Telegram Chat', 'telegramChat', 'https://t.me/completeclubchat'],
    ['YouTube', 'youtube', 'https://youtube.com/@completeclub'],
    ['TikTok', 'tiktok', 'https://tiktok.com/@completeclub'],
    ['LinkedIn', 'linkedin', 'https://linkedin.com/company/completeclub']
  ] as const
  for (const [label, , value] of links) await preview.getByLabel(label, { exact: true }).fill(value)
  await preview.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Complete club edit', exact: true })).toBeVisible()
  const saved = (await updates(page))[0]
  expect(saved).toMatchObject({
    id: 20,
    title: 'Complete club edit',
    organizationTypeId: 2,
    ...Object.fromEntries(links.map(([, field, value]) => [field, value]))
  })
  expect(saved.description).toContain('Updated description from the club editor')
  await expect(preview.locator('.club-preview-properties')).toContainText('Community')
  await expect(preview.locator('.club-preview-properties')).toContainText('Frozen')
  await expect(page.getByTestId('overview-links')).toContainText('7')

  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  for (const [label] of links) await preview.getByLabel(label, { exact: true }).fill('')
  await description.click()
  await page.keyboard.press('ControlOrMeta+A')
  await page.keyboard.press('Backspace')
  await expect(preview.getByText('0/2000', { exact: true })).toBeVisible()
  await preview.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect.poll(async () => (await updates(page)).length).toBe(2)
  const cleared = (await updates(page))[1]
  expect(cleared.title).toBeUndefined()
  for (const [, field] of links) expect(cleared[field]).toBe('')
  expect(JSON.parse(String(cleared.description))).toEqual([{ type: 'p', children: [{ text: '' }] }])
  await expect(page.getByTestId('overview-links')).toContainText('0')
})

test('opening an unchanged editor does not create an unsaved draft', async ({ page }) => {
  await page.goto('/admin')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Campus Club')
  await expect(page.locator('[contenteditable="true"]')).toContainText('Campus technology community')
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toHaveCount(0)
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
  expect(await updates(page)).toHaveLength(0)
})

test('success notifications leave subsequent save controls clickable', async ({ page }) => {
  await page.goto('/admin')
  for (const name of ['First club edit', 'Second club edit']) {
    await page.getByRole('button', { name: 'Edit club', exact: true }).click()
    await page.getByLabel('Name', { exact: true }).fill(name)
    await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await expect(page.getByRole('heading', { name, exact: true })).toBeVisible()
  }
  expect(await updates(page)).toHaveLength(2)
})

test('an unsaved edit survives full page, popup, minimize and restore', async ({ page }) => {
  await page.goto('/admin')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Draft across layouts')
  for (const layout of ['Full Page', 'Pop Up', 'Minimize']) {
    await page.getByRole('button', { name: 'Switch layout', exact: true }).click()
    await page.getByRole('menuitemradio', { name: layout, exact: true }).click()
    if (layout === 'Minimize') {
      await expect(page.getByLabel('Name', { exact: true })).toBeHidden()
      await page.getByRole('button', { name: 'Restore Campus Club preview', exact: true }).click()
    }
    await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Draft across layouts')
  }
  await page.keyboard.press('Escape')
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Draft across layouts')
  expect(await updates(page)).toHaveLength(0)
})

test('opening and closing preserves table search, sort, column visibility and page scroll', async ({ page }) => {
  await page.goto('/admin?closed')
  const table = page.getByRole('table', { name: 'clubs data table', exact: true })
  const search = page.getByRole('textbox', { name: 'Search clubs...', exact: true })
  await search.fill('Design')
  const header = page.getByRole('columnheader', { name: 'Organization', exact: true })
  await header.click()
  await expect(header).toHaveAttribute('aria-sort', 'ascending')
  await page.getByRole('button', { name: 'View', exact: true }).click()
  await page.getByRole('menuitemcheckbox', { name: 'Status', exact: true }).click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('columnheader', { name: 'Status', exact: true })).toHaveCount(0)
  await page.addStyleTag({ content: 'main { min-height: 180vh; }' })
  await page.evaluate(() => window.scrollTo(0, 200))
  const scrollBefore = await page.evaluate(() => window.scrollY)
  expect(scrollBefore).toBe(200)
  await table.getByRole('cell', { name: 'Design Club', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Design Club', exact: true })).toBeVisible()
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(scrollBefore)
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(scrollBefore)
  await expect(search).toHaveValue('Design')
  await expect(header).toHaveAttribute('aria-sort', 'ascending')
  await expect(table.locator('tbody tr')).toHaveCount(1)
  await expect(page.getByRole('columnheader', { name: 'Status', exact: true })).toHaveCount(0)
})

test('reference geometry and both theme colors remain correct with reduced motion', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.setViewportSize({ width: 1440, height: 1080 })
  await page.goto('/admin')
  const preview = page.getByTestId('club-preview')
  await expect(preview).toHaveCSS('background-color', 'rgb(10, 10, 10)')
  await expect(preview.locator('.club-preview-topbar')).toHaveCSS('height', '48px')
  await expect(preview.locator('.club-preview-tabs')).toHaveCSS('height', '48px')
  await expect(preview.locator('.club-preview-logo')).toHaveCSS('width', '60px')
  await expect(preview.locator('.club-preview-logo')).toHaveCSS('height', '60px')
  await expect(preview.getByRole('heading', { name: 'Campus Club', exact: true })).toHaveCSS('font-size', '20px')
  await expect(preview.getByRole('heading', { name: 'Campus Club', exact: true })).toHaveCSS('line-height', '28px')
  const styles = await preview.evaluate((element) => {
    const computed = getComputedStyle(element)
    return { fontFamily: computed.fontFamily, duration: computed.transitionDuration }
  })
  expect(styles.fontFamily).toContain('Inter')
  for (const duration of styles.duration.split(',')) expect(Number.parseFloat(duration)).toBeLessThan(0.001)
  await page.evaluate(() => document.documentElement.classList.remove('dark'))
  await expect(preview).toHaveCSS('background-color', 'rgb(255, 255, 255)')
})
