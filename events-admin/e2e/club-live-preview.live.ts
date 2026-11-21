import { readFile } from 'node:fs/promises'
// Opt-in integration suite: requires the isolated local Docker QA services and generated test accounts.
import { type BrowserContext, expect, type Page, test } from '@playwright/test'
import { Capability, OrganizationStatus } from '@studyverse/proto'

type Account = { id: number; email: string; password: string }
type Organization = {
  id: number
  title: string
  description?: string
  organizationTypeId: number
  imageUrl?: string
  [field: string]: unknown
}
type Role = { id: number; displayName: string; parentTemplateId?: number; maxHolders?: number; isActive?: boolean }
let accounts: Record<string, Account>

async function login(context: BrowserContext, name = 'admin') {
  const account = accounts[name]
  const flowResponse = await context.request.get('http://localhost:14433/self-service/login/browser', {
    headers: { Accept: 'application/json' }
  })
  expect(flowResponse.ok()).toBe(true)
  const flow: { ui: { action: string; nodes: { attributes: { name?: string; value?: string } }[] } } =
    await flowResponse.json()
  const csrf = flow.ui.nodes.find((node) => node.attributes.name === 'csrf_token')?.attributes.value
  const response = await context.request.post(flow.ui.action, {
    headers: { Accept: 'application/json' },
    data: { method: 'password', identifier: account.email, password: account.password, csrf_token: csrf }
  })
  expect(response.status(), 'Real Kratos browser login must succeed').toBe(200)
}

async function rpc<T>(context: BrowserContext, service: string, method: string, data: object): Promise<T> {
  const response = await context.request.post(`http://localhost:5555/events.v1.${service}/${method}`, {
    headers: { 'Connect-Protocol-Version': '1' },
    data
  })
  const body = await response.json()
  expect(response.ok(), `${service}/${method}: ${response.status()} ${body.message || ''}`).toBe(true)
  return body as T
}

async function createClub(context: BrowserContext, suffix: string, typeId?: number) {
  const title = `QA Live ${suffix} ${Date.now()}`
  const type = typeId
    ? { id: typeId, title: '' }
    : (
        await rpc<{ organizationType: { id: number; title: string } }>(
          context,
          'OrganizationTypesService',
          'CreateOrganizationType',
          { title: `QA Type ${suffix} ${Date.now()}` }
        )
      ).organizationType
  const { organization } = await rpc<{ organization: Organization }>(
    context,
    'OrganizationsService',
    'CreateOrganization',
    { title, organizationTypeId: type.id, description: '[{"type":"p","children":[{"text":"QA description"}]}]' }
  )
  return { organization, type }
}

async function openClub(page: Page, title: string, query: Record<string, string> = {}) {
  await page.goto(`/administration/organizations?${new URLSearchParams({ search: title, ...query })}`)
  await page.getByRole('cell').filter({ hasText: title }).click()
  await expect(page.getByTestId('club-preview')).toBeVisible()
}

async function createRole(context: BrowserContext, typeId: number, data: object) {
  return (
    await rpc<{ roleTemplate: Role }>(context, 'RoleTemplatesService', 'CreateRoleTemplate', {
      organizationTypeId: typeId,
      isActive: true,
      ...data
    })
  ).roleTemplate
}

test.beforeAll(async () => {
  accounts = JSON.parse(
    await readFile(new URL('../test-results/club-preview-qa/accounts.json', import.meta.url), 'utf8')
  )
})

test.beforeEach(async ({ context }) => login(context))

test('the production error handler keeps save failures inline and the draft can be retried', async ({
  page,
  context
}) => {
  const { organization } = await createClub(context, 'Save failure')
  await page.setViewportSize({ width: 1280, height: 720 })
  await openClub(page, organization.title)
  await page.getByRole('button', { name: 'Edit club', exact: true }).click()
  const draftTitle = `${organization.title} edited`
  await page.getByLabel('Name', { exact: true }).fill(draftTitle)
  const endpoint = '**/events.v1.OrganizationsService/UpdateOrganization'
  await page.route(endpoint, (route) =>
    route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({ code: 'unavailable', message: 'QA save interruption' })
    })
  )
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByTestId('club-preview').getByRole('alert')).toContainText('QA save interruption')
  await expect(page.getByLabel('Name', { exact: true })).toHaveValue(draftTitle)
  await expect(page.locator('[data-sonner-toast][data-type="error"]')).toHaveCount(0)
  await page.unroute(endpoint)
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByRole('heading', { name: draftTitle, exact: true })).toBeVisible()
  const saved = await rpc<{ organization: Organization }>(context, 'OrganizationsService', 'GetOrganization', {
    id: organization.id
  })
  expect(saved.organization.title).toBe(draftTitle)
})

test('real browser, backend and database persist all fields, an uploaded logo and cleared values', async ({
  page,
  context
}) => {
  const { organization, type } = await createClub(context, 'Fields')
  const { organizationType: otherType } = await rpc<{ organizationType: { id: number; title: string } }>(
    context,
    'OrganizationTypesService',
    'CreateOrganizationType',
    { title: `QA Alternate ${Date.now()}` }
  )
  await openClub(page, organization.title, {
    type: String(type.id),
    status: String(OrganizationStatus.ACTIVE),
    sortBy: 'title',
    sortDesc: '1'
  })
  const preview = page.getByTestId('club-preview')
  await preview.getByRole('button', { name: 'Edit club', exact: true }).click()
  const editedTitle = `${organization.title} edited`
  await preview.getByLabel('Name', { exact: true }).fill(editedTitle)
  const editor = preview.locator('[contenteditable="true"]')
  await editor.click()
  await editor.press('ControlOrMeta+A')
  await page.keyboard.insertText('Persisted through the actual backend')
  await preview.getByRole('combobox', { name: 'Type', exact: true }).click()
  await page.getByRole('option', { name: otherType.title, exact: true }).click()
  await preview.getByRole('combobox', { name: 'Status', exact: true }).click()
  await page.getByRole('option', { name: 'Frozen', exact: true }).click()
  const links = [
    ['Website', 'website', 'https://club.example.test'],
    ['Instagram', 'instagram', 'https://instagram.com/liveclub'],
    ['Telegram Channel', 'telegramChannel', 'https://t.me/liveclub'],
    ['Telegram Chat', 'telegramChat', 'https://t.me/liveclubchat'],
    ['YouTube', 'youtube', 'https://youtube.com/@liveclub'],
    ['TikTok', 'tiktok', 'https://tiktok.com/@liveclub'],
    ['LinkedIn', 'linkedin', 'https://linkedin.com/company/liveclub']
  ] as const
  for (const [label, , value] of links) await preview.getByLabel(label, { exact: true }).fill(value)
  const png = await page.evaluate(() => {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 128
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas is unavailable')
    context.fillStyle = '#8564d9'
    context.fillRect(0, 0, 128, 128)
    return canvas.toDataURL('image/png').split(',')[1]
  })
  await page.locator('input[type=file]').setInputFiles({
    name: 'live-logo.png',
    mimeType: 'image/png',
    buffer: Buffer.from(png, 'base64')
  })
  const crop = page.getByRole('dialog', { name: 'Edit Image', exact: true })
  await expect
    .poll(() => crop.locator('img').evaluateAll((images) => images.some((img) => img.complete && img.naturalWidth > 0)))
    .toBe(true)
  await crop.getByRole('button', { name: 'Apply', exact: true }).click()
  await preview.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(preview.getByRole('heading', { name: editedTitle, exact: true })).toBeVisible({ timeout: 20_000 })
  expect(new URL(page.url()).searchParams.get('type')).toBe(String(type.id))
  expect(new URL(page.url()).searchParams.get('status')).toBe(String(OrganizationStatus.ACTIVE))
  expect(new URL(page.url()).searchParams.get('sortDesc')).toBe('1')
  const saved = (
    await rpc<{ organization: Organization }>(context, 'OrganizationsService', 'GetOrganization', {
      id: organization.id
    })
  ).organization
  expect(saved.organizationTypeId).toBe(otherType.id)
  expect(saved.status).toContain('FROZEN')
  expect(saved.description).toContain('Persisted through the actual backend')
  for (const [, field, value] of links) expect(saved[field]).toBe(value)
  expect(saved.imageUrl).toContain('http://localhost:19000/preview-images/')
  expect((await context.request.get(String(saved.imageUrl))).status()).toBe(200)
  await expect(page.getByTestId('overview-links')).toContainText('7')
  await expect(preview.locator('.club-preview-logo img')).toBeVisible()
  await page.screenshot({ path: test.info().outputPath('club-live-dark.png') })
  await page.evaluate(() => document.documentElement.classList.remove('dark'))
  await page.screenshot({ path: test.info().outputPath('club-live-light.png') })

  await openClub(page, editedTitle)
  await expect(preview.getByRole('heading', { name: editedTitle, exact: true })).toBeVisible()
  await preview.getByRole('button', { name: 'Edit club', exact: true }).click()
  for (const [label] of links) await preview.getByLabel(label, { exact: true }).fill('')
  await editor.click()
  await editor.press('ControlOrMeta+A')
  await editor.press('Backspace')
  await preview.getByRole('button', { name: 'Remove image', exact: true }).click()
  await preview.getByRole('combobox', { name: 'Type', exact: true }).click()
  await page.getByRole('option', { name: type.title, exact: true }).click()
  await preview.getByRole('combobox', { name: 'Status', exact: true }).click()
  await page.getByRole('option', { name: 'Active', exact: true }).click()
  await preview.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByTestId('overview-links')).toContainText('0')
  const cleared = (
    await rpc<{ organization: Organization }>(context, 'OrganizationsService', 'GetOrganization', {
      id: organization.id
    })
  ).organization
  expect(cleared.organizationTypeId).toBe(type.id)
  expect(cleared.status).toContain('ACTIVE')
  expect(cleared.imageUrl || '').toBe('')
  for (const [, field] of links) expect(cleared[field] || '').toBe('')
  expect(JSON.parse(String(cleared.description))).toEqual([{ type: 'p', children: [{ text: '' }] }])
})

test('real team and role edits clear database values and shared changes appear in another club', async ({
  page,
  context
}) => {
  const { organization, type } = await createClub(context, 'Roles')
  const second = await createClub(context, 'Second', type.id)
  const shared = await createRole(context, type.id, { key: 'president', displayName: 'QA President' })
  const local = await createRole(context, type.id, {
    organizationId: organization.id,
    key: 'coordinator',
    displayName: 'QA Coordinator',
    parentTemplateId: shared.id,
    maxHolders: 5,
    capabilities: [Capability.CREATE_EVENT]
  })
  await openClub(page, organization.title)
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await page.getByRole('button', { name: 'Add Member', exact: true }).click()
  const add = page.getByRole('dialog', { name: 'Add Organization Member', exact: true })
  await add.getByLabel('Search User', { exact: true }).fill(`#${accounts.reader.id}`)
  await add.getByRole('button', { name: 'Find user', exact: true }).click()
  await add.getByRole('button', { name: /Reader Preview QA/ }).click()
  await add.getByRole('combobox', { name: 'Role', exact: true }).click()
  await page.getByRole('option', { name: 'QA Coordinator', exact: true }).click()
  await add.getByLabel('Member until (optional)', { exact: true }).fill('2035-06-01T12:00')
  await add.getByRole('button', { name: 'Add Member', exact: true }).click()
  await page.getByRole('button', { name: 'Edit membership for Reader Preview QA', exact: true }).click()
  await page.getByLabel('Member until (optional)', { exact: true }).fill('')
  await page.getByRole('button', { name: 'Save membership', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Edit membership for Reader Preview QA', exact: true })).toBeVisible()
  const members = await rpc<{ members: { roleTemplateId: number; validUntil?: string }[] }>(
    context,
    'OrganizationMembersService',
    'ListOrganizationMembers',
    { organizationId: organization.id }
  )
  expect(members.members[0].roleTemplateId).toBe(local.id)
  expect(members.members[0].validUntil || '').toBe('')
  await page.getByRole('button', { name: 'Edit Roles', exact: true }).click()
  const roles = page.getByRole('dialog', { name: 'Role Structure Editor', exact: true })
  await roles.getByText('QA Coordinator', { exact: true }).first().click()
  await roles.getByRole('combobox', { name: 'Reports To', exact: true }).click()
  await page.getByRole('option', { name: 'No parent (top level)', exact: true }).click()
  await roles.getByLabel('Max Members', { exact: true }).fill('')
  await roles.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect
    .poll(async () => {
      const response = await rpc<{ roleTemplates: Role[] }>(context, 'RoleTemplatesService', 'ListRoleTemplatesByOrg', {
        organizationId: organization.id,
        organizationTypeId: type.id
      })
      const role = response.roleTemplates.find((entry) => entry.id === local.id)
      return { parent: role?.parentTemplateId, limit: role?.maxHolders }
    })
    .toEqual({ parent: undefined, limit: undefined })
  await roles.getByText('QA President', { exact: true }).first().click()
  await expect(roles.getByText(/^Applies to all organizations of this type\./)).toBeVisible()
  await roles.getByLabel('Display Name', { exact: true }).fill('QA Shared Lead')
  await roles.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(roles.getByText('QA Shared Lead', { exact: true }).first()).toBeVisible()
  await roles.getByRole('button', { name: 'Close', exact: true }).click()
  await page.getByRole('tab', { name: 'History', exact: true }).click()
  await expect(page.locator('.club-preview-history')).not.toContainText('No membership changes recorded yet')
  await expect(page.locator('.club-preview-history')).toContainText('Reader')
  await openClub(page, second.organization.title)
  await page.getByRole('tab', { name: 'Team', exact: true }).click()
  await page.getByRole('button', { name: 'Edit Roles', exact: true }).click()
  await expect(
    page
      .getByRole('dialog', { name: 'Role Structure Editor', exact: true })
      .getByText('QA Shared Lead', { exact: true })
      .first()
  ).toBeVisible()
})

test('real permissions separate member managers, settings managers and readonly users', async ({
  browser,
  context
}) => {
  const { organization, type } = await createClub(context, 'Permissions')
  const shared = await createRole(context, type.id, { key: 'common', displayName: 'QA Common Role' })
  for (const [name, capability] of [
    ['settings', Capability.MANAGE_SETTINGS],
    ['members', Capability.MANAGE_MEMBERS]
  ] as const) {
    const role = await createRole(context, type.id, {
      organizationId: organization.id,
      key: name,
      displayName: `QA ${name}`,
      capabilities: [capability]
    })
    await rpc(context, 'OrganizationMembersService', 'AddOrganizationMember', {
      organizationId: organization.id,
      userId: accounts[name].id,
      roleTemplateId: role.id
    })
  }
  for (const name of ['settings', 'members', 'reader']) {
    const userContext = await browser.newContext()
    try {
      await login(userContext, name)
      const userPage = await userContext.newPage()
      await openClub(userPage, organization.title)
      await expect(userPage.getByRole('button', { name: 'Edit club', exact: true })).toHaveCount(
        name === 'settings' ? 1 : 0
      )
      await userPage.getByRole('tab', { name: 'Team', exact: true }).click()
      await expect(userPage.getByRole('button', { name: 'Add Member', exact: true })).toHaveCount(
        name === 'members' ? 1 : 0
      )
      const denied = await userContext.request.post(
        'http://localhost:5555/events.v1.RoleTemplatesService/UpdateRoleTemplate',
        { data: { id: shared.id, displayName: 'Unauthorized change' } }
      )
      expect(denied.status()).toBe(403)
      const endpoint =
        name === 'settings'
          ? 'OrganizationMembersService/AddOrganizationMember'
          : 'OrganizationsService/UpdateOrganization'
      const input =
        name === 'settings'
          ? { organizationId: organization.id, userId: accounts.admin.id, roleTemplateId: shared.id }
          : { id: organization.id, title: 'Unauthorized change' }
      expect(
        (await userContext.request.post(`http://localhost:5555/events.v1.${endpoint}`, { data: input })).status()
      ).toBe(403)
    } finally {
      await userContext.close()
    }
  }
})
