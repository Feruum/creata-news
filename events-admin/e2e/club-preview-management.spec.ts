import { expect, type Page, test } from '@playwright/test'

async function requests(page: Page, method: string) {
  const log: { method: string; input: Record<string, unknown> }[] = JSON.parse(
    (await page.getByTestId('requests').textContent()) || '[]'
  )
  return log.filter((entry) => entry.method === method).map((entry) => entry.input)
}

test('keyboard bulk selection does not open or change the club preview', async ({ page }) => {
  await page.goto('/admin?closed')
  const checkbox = page.getByRole('checkbox', { name: 'Select row', exact: true }).first()
  await checkbox.focus()
  await page.keyboard.press('Space')
  await expect(checkbox).toBeChecked()
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
  await expect(page.locator('tr[data-preview-active=true]')).toHaveCount(0)
})

test('switching rows during a save cannot discard or detach the pending form', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto('/admin?slowSave')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Saved Campus Club')
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Save Changes', exact: true })).toBeDisabled()
  await page.getByRole('cell', { name: 'Design Club', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toHaveCount(0)
  await expect(page.locator('tr[data-preview-active=true]')).toContainText('Campus Club')
  await expect(page.getByRole('heading', { name: 'Saved Campus Club', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByTestId('club-preview')).toHaveCount(0)
})

test('overview refresh fetches real counts again', async ({ page }) => {
  await page.goto('/admin')
  await expect(page.getByTestId('overview-subscribers')).toContainText('48')
  const before = (await requests(page, 'GetOrganizationSubscriberCount')).length
  await page.getByRole('button', { name: 'Refresh', exact: true }).click()
  await expect.poll(async () => (await requests(page, 'GetOrganizationSubscriberCount')).length).toBeGreaterThan(before)
})

test('a background settings refetch failure preserves the editing draft', async ({ page }) => {
  await page.goto('/admin?typesAfterError')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await page.getByLabel('Name', { exact: true }).fill('Keep this draft')
  await page.getByRole('button', { name: 'More club actions', exact: true }).click()
  await page.getByRole('menuitem', { name: 'Refresh club', exact: true }).click()
  await expect.poll(async () => (await requests(page, 'ListOrganizationTypes')).length).toBeGreaterThan(1)
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Keep this draft')
  await page.getByRole('button', { name: 'Close preview', exact: true }).click()
  await expect(page.getByRole('alertdialog')).toBeVisible()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Keep this draft')
})

test('load, settings and history failures remain inside the panel', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.goto('/admin?loadError')
  await expect(page.getByTestId('club-preview').getByRole('alert')).toContainText('Could not load club')
  await expect(page.getByRole('button', { name: 'Close preview', exact: true })).toBeVisible()
  await page.goto('/admin?typesError')
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  await expect(page.getByTestId('club-preview').getByRole('alert')).toContainText(
    'Could not load organization settings'
  )
  await page.goto('/admin?historyError')
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await expect(page.getByTestId('club-preview').getByRole('alert')).toContainText('Could not load membership history')
  await expect(page.getByText('No membership changes recorded yet', { exact: true })).toHaveCount(0)
  expect(errors).toEqual([])
})

test('role editing clears parent and limit, stores active state and can clear all capabilities', async ({ page }) => {
  await page.goto('/admin')
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await page.getByRole('button', { name: 'Edit Roles', exact: true }).click()
  const editor = page.getByRole('dialog', { name: 'Role Structure Editor', exact: true })
  await editor.getByText('Coordinator', { exact: true }).first().click()
  await editor.getByRole('combobox', { name: 'Reports To', exact: true }).click()
  await page.getByRole('option', { name: 'No parent (top level)', exact: true }).click()
  await editor.getByLabel('Max Members', { exact: true }).fill('')
  await editor.getByLabel('Sort Order', { exact: true }).fill('4')
  await editor.getByRole('switch', { name: 'Active role', exact: true }).uncheck()
  await editor.getByRole('checkbox', { name: 'Create Events', exact: true }).uncheck()
  await editor.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect.poll(async () => (await requests(page, 'SetTemplateCapabilities')).length).toBe(1)
  expect((await requests(page, 'UpdateRoleTemplate'))[0]).toMatchObject({
    id: 7,
    clearParentTemplate: true,
    clearMaxHolders: true,
    sortOrder: 4,
    isActive: false
  })
  expect((await requests(page, 'SetTemplateCapabilities'))[0]).toMatchObject({ roleTemplateId: 7, capabilities: [] })
})

test('shared roles show global scope and stay readonly for a club-only settings manager', async ({ page }) => {
  for (const [query, editable] of [
    ['', true],
    ['?settingsOnly', false]
  ] as const) {
    await page.goto(`/admin${query}`)
    await page.getByRole('tab', { name: 'Team', exact: true }).click()
    await page.getByRole('button', { name: 'Edit Roles', exact: true }).click()
    const editor = page.getByRole('dialog', { name: 'Role Structure Editor', exact: true })
    await editor.getByText('President', { exact: true }).first().click()
    await expect(editor.getByText('Shared role template', { exact: true }).first()).toBeVisible()
    await expect(editor.getByText(/^Applies to all organizations of this type\./)).toBeVisible()
    if (editable) await expect(editor.getByLabel('Display Name', { exact: true })).toBeEnabled()
    else await expect(editor.getByLabel('Display Name', { exact: true })).toBeDisabled()
  }
})

test('members can be added by ID, have expiry cleared, and be removed', async ({ page }) => {
  await page.goto('/admin?empty')
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await page.getByRole('button', { name: 'Add Member', exact: true }).click()
  const add = page.getByRole('dialog', { name: 'Add Organization Member', exact: true })
  await add.getByLabel('Search User', { exact: true }).fill('#142')
  await add.getByRole('button', { name: 'Find user', exact: true }).click()
  await add.getByRole('button', { name: /Madi Talgat/ }).click()
  await add.getByRole('combobox', { name: 'Role', exact: true }).click()
  await page.getByRole('option', { name: 'Coordinator', exact: true }).click()
  await add.getByLabel('Member until (optional)', { exact: true }).fill('2035-06-01T12:00')
  await add.getByRole('button', { name: 'Add Member', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Edit membership for Madi Talgat', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Edit membership for Madi Talgat', exact: true }).click()
  await page.getByLabel('Member until (optional)', { exact: true }).fill('')
  await page.getByRole('button', { name: 'Save membership', exact: true }).click()
  await expect.poll(async () => (await requests(page, 'UpdateOrganizationMember')).length).toBe(1)
  expect((await requests(page, 'UpdateOrganizationMember'))[0]).toMatchObject({
    id: 15,
    roleTemplateId: 7,
    validUntil: ''
  })
  await page.getByRole('button', { name: 'Remove Madi Talgat', exact: true }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Remove member', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'No members yet', exact: true })).toBeVisible()
})

test('logo replacement uploads a real file and upload errors preserve editing', async ({ page }) => {
  for (const fail of [false, true]) {
    await page.goto(fail ? '/admin?uploadError' : '/admin')
    await page.getByRole('button', { name: 'Edit club', exact: true }).click()
    const logoPng = await page.evaluate(() => {
      const canvas = document.createElement('canvas')
      canvas.width = 128
      canvas.height = 128
      const context = canvas.getContext('2d')
      if (!context) throw new Error('Canvas is unavailable')
      context.fillStyle = '#8564d9'
      context.fillRect(0, 0, 128, 128)
      return canvas.toDataURL('image/png').split(',')[1]
    })
    await page.locator('input[type=file]').setInputFiles({
      name: 'club-logo.png',
      mimeType: 'image/png',
      buffer: Buffer.from(logoPng, 'base64')
    })
    const crop = page.getByRole('dialog', { name: 'Edit Image', exact: true })
    await expect(crop).toBeVisible()
    await expect
      .poll(async () =>
        crop.locator('img').evaluateAll((images) => images.some((image) => image.complete && image.naturalWidth > 0))
      )
      .toBe(true)
    await crop.getByRole('button', { name: 'Apply', exact: true }).click()
    await expect(crop).toHaveCount(0)
    await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
    await expect.poll(async () => (await requests(page, 'UploadOrganizationImage')).length).toBe(1)
    if (fail) {
      await expect(page.getByTestId('club-preview').getByRole('alert')).toHaveText(
        'Could not connect to the server. Check your connection and try again.'
      )
      await expect(page.getByLabel('Name', { exact: true })).toHaveValue('Campus Club')
      expect(await requests(page, 'UpdateOrganization')).toHaveLength(0)
    } else {
      await expect.poll(async () => (await requests(page, 'UpdateOrganization')).length).toBe(1)
      expect((await requests(page, 'UpdateOrganization'))[0]).toMatchObject({ imageUrl: '/club-avatar.svg' })
    }
  }
})
