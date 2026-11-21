import { expect, type Page, test } from '@playwright/test'

async function requests(page: Page, method: string) {
  const log: { method: string; input: Record<string, unknown> }[] = JSON.parse(
    (await page.getByTestId('requests').textContent()) || '[]'
  )
  return log.filter((request) => request.method === method).map((request) => request.input)
}

test('club preview opens as a nonmodal drawer with real overview data', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1080 })
  await page.goto('/admin')
  const preview = page.getByTestId('club-preview')
  await expect(preview).toBeVisible()
  await expect(preview).toHaveAttribute('data-layout', 'drawer')
  await expect(page.getByRole('heading', { name: 'Campus Club', exact: true })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Details', exact: true })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByTestId('overview-subscribers')).toContainText('48')
  const bounds = await preview.boundingBox()
  expect(bounds?.x).toBeGreaterThan(500)
  expect(bounds?.width).toBeGreaterThanOrEqual(600)
  await expect(page.locator('[data-slot="dialog-overlay"]')).toHaveCount(0)
  await page.screenshot({ path: test.info().outputPath('club-preview-dark.png'), fullPage: true })
  await page.evaluate(() => document.documentElement.classList.remove('dark'))
  await expect(preview).toHaveCSS('background-color', 'rgb(255, 255, 255)')
  await page.screenshot({ path: test.info().outputPath('club-preview-light.png'), fullPage: true })
})

test('all layouts retain the club and selected tab and drawer supports resizing', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/admin')
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  for (const [label, layout] of [
    ['Full Page', 'full-page'],
    ['Pop Up', 'pop-up'],
    ['Minimize', 'minimize'],
    ['Drawer', 'drawer']
  ]) {
    await page.getByRole('button', { name: 'Switch layout', exact: true }).click()
    await page.getByRole('menuitemradio', { name: label, exact: true }).click()
    await expect(page.getByTestId('club-preview')).toHaveAttribute('data-layout', layout)
    if (layout !== 'minimize')
      await expect(page.getByRole('tab', { name: 'Team', exact: true })).toHaveAttribute('aria-selected', 'true')
  }
  const before = await page.getByTestId('club-preview').boundingBox()
  const handle = page.getByRole('separator', { name: 'Resize club preview', exact: true })
  await handle.focus()
  await page.keyboard.press('ArrowLeft')
  await expect
    .poll(async () => (await page.getByTestId('club-preview').boundingBox())?.width)
    .toBeGreaterThan(before?.width || 0)
  const box = await handle.boundingBox()
  if (!box) throw new Error('Resize handle is missing')
  await page.mouse.move(box.x + box.width / 2, box.y + 100)
  await page.mouse.down()
  await page.mouse.move(box.x - 90, box.y + 100, { steps: 8 })
  await page.mouse.up()
  await expect
    .poll(async () => (await page.getByTestId('club-preview').boundingBox())?.width)
    .toBeGreaterThan((before?.width || 0) + 60)
})

test('edit stays in the panel, clears values and saves without creating another club', async ({ page }) => {
  await page.goto('/admin')
  const preview = page.getByTestId('club-preview')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(1)
  await preview.getByLabel('Instagram', { exact: true }).fill('')
  await preview.getByLabel('Website', { exact: true }).fill('https://campusclub.example.test')
  await preview.getByRole('button', { name: 'Remove image', exact: true }).click()
  await preview.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect.poll(async () => (await requests(page, 'UpdateOrganization')).length).toBe(1)
  const input = (await requests(page, 'UpdateOrganization'))[0]
  expect(input).toMatchObject({ id: 20, instagram: '', imageUrl: '', website: 'https://campusclub.example.test' })
  expect(input.title).toBeUndefined()
  expect(await requests(page, 'CreateOrganization')).toHaveLength(0)
  await expect(preview.getByRole('tab', { name: 'Details', exact: true })).toBeVisible()
  await expect(preview.getByText('campusclub.example.test', { exact: true }).first()).toBeVisible()
})

test('cancel discards form changes and closing a dirty preview requires a choice', async ({ page }) => {
  await page.goto('/admin')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Unsaved name')
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Unsaved name')
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Campus Club', exact: true })).toBeVisible()
  expect(await requests(page, 'UpdateOrganization')).toHaveLength(0)
})

test('failed save keeps the draft and retry updates the existing club', async ({ page }) => {
  await page.goto('/admin?saveError=once')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Updated Campus Club')
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Updated Campus Club')
  await expect(page.getByTestId('club-preview').getByRole('alert')).toHaveText(
    'Could not connect to the server. Check your connection and try again.'
  )
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Updated Campus Club', exact: true })).toBeVisible()
})

test('readonly, member managers and settings managers have separate controls', async ({ page }) => {
  await page.goto('/admin?readonly')
  await expect(page.getByRole('button', { name: 'Edit club', exact: true })).toHaveCount(0)
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Add Member', exact: true })).toHaveCount(0)
  await page.goto('/admin?membersOnly')
  await expect(page.getByRole('button', { name: 'Edit club', exact: true })).toHaveCount(0)
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Add Member', exact: true })).toBeVisible()
  await page.goto('/admin?settingsOnly')
  await expect(page.getByRole('button', { name: 'Edit club', exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Add Member', exact: true })).toHaveCount(0)
})

test('discarding a draft opens the requested club and keeps row selection independent', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/admin')
  await expect(page.locator('tr[data-preview-active=true]')).toContainText('Campus Club')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Unsaved club')
  await page.getByRole('cell', { name: 'Design Club', exact: true }).click()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Unsaved club')
  await page.getByRole('cell', { name: 'Design Club', exact: true }).click()
  await page.getByRole('button', { name: 'Discard changes', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Design Club', exact: true })).toBeVisible()
  await expect(page.locator('tr[data-preview-active=true]')).toContainText('Design Club')
  expect(await requests(page, 'UpdateOrganization')).toHaveLength(0)
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
})

test('closing with Escape restores the selected table row focus', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/admin?closed')
  const row = page.getByRole('row').filter({ has: page.getByRole('cell', { name: 'Design Club', exact: true }) })
  await row.getByRole('cell', { name: 'Design Club', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Design Club', exact: true })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
  await expect(row).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { name: 'Design Club', exact: true })).toBeVisible()
})

test('mobile layouts and editing fit the viewport, with visible save controls', async ({ page }) => {
  for (const width of [320, 390, 768]) {
    await page.setViewportSize({ width, height: 844 })
    await page.goto('/admin')
    const bounds = await page.getByTestId('club-preview').boundingBox()
    expect(bounds?.width).toBe(width)
    await page.getByRole('button', { name: 'Edit club', exact: true }).click()
    await page.getByLabel('TikTok', { exact: true }).scrollIntoViewIfNeeded()
    await expect(page.getByRole('button', { name: 'Save Changes', exact: true })).toBeInViewport()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    if (width === 390) await page.screenshot({ path: test.info().outputPath('club-preview-mobile-edit.png') })
  }
})
