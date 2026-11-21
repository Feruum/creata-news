import { expect, type Page, test } from '@playwright/test'

async function mockOrganizations(
  page: Page,
  options: { staff?: boolean; administrator?: boolean; failOnce?: boolean } = {}
) {
  const baseURL = test.info().project.use.baseURL
  if (!baseURL) throw new Error('Playwright baseURL is required')
  const appOrigin = new URL(baseURL).origin
  let exists = true
  const deleted: number[] = []
  await page.route('**/*', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = url.pathname.split('/').at(-1) ?? ''
    const headers = {
      'access-control-allow-origin': appOrigin,
      'access-control-allow-credentials': 'true'
    }
    if (url.pathname.endsWith('/sessions/whoami')) {
      return route.fulfill({
        headers,
        json: {
          id: 'organization-deletion-session',
          active: true,
          identity: { id: 'organization-deletion-identity', traits: { email: 'staff@example.test' } }
        }
      })
    }
    if (method === 'DeleteOrganization') {
      const body: { id: number } = request.postDataJSON()
      deleted.push(body.id)
      if (options.failOnce && deleted.length === 1) {
        return route.fulfill({
          status: 503,
          headers,
          json: { code: 'unavailable', message: 'Temporarily unavailable' }
        })
      }
      exists = false
      return route.fulfill({ headers, json: { success: true } })
    }
    const payloads: Record<string, unknown> = {
      GetMe: {
        user: { id: 1, username: 'organization-qa', email: 'staff@example.test' },
        permissions: { isStaff: !!options.staff, isAdmin: !!options.administrator, managedClubIds: [2] }
      },
      ListOrganizations: {
        organizations: [
          ...(exists ? [{ id: 2, title: 'Fixture QA club', organizationTypeId: 1 }] : []),
          { id: 3, title: 'Community club', organizationTypeId: 1 }
        ],
        total: exists ? 2 : 1
      },
      ListOrganizationTypes: { organizationTypes: [{ id: 1, title: 'Club' }], total: 1 },
      GlobalSearch: { results: [], totalHits: 0 }
    }
    if (method in payloads) return route.fulfill({ headers, json: payloads[method] })
    if (url.origin === appOrigin) {
      return route.continue()
    }
    return route.abort()
  })
  return deleted
}

for (const role of ['staff', 'administrator'] as const) {
  test(`${role} deletes only the confirmed organization and refreshes the server list`, async ({ page }) => {
    const deleted = await mockOrganizations(page, { [role]: true })
    await page.goto('/organizations')
    const row = page.getByRole('row').filter({ hasText: 'Fixture QA club' })
    await row.getByRole('button', { name: 'Delete row', exact: true }).click()
    await expect(page.getByRole('alertdialog')).toContainText('cannot be undone')
    expect(deleted).toEqual([])
    await page.getByRole('button', { name: 'Cancel', exact: true }).click()
    await expect(row).toBeVisible()
    expect(deleted).toEqual([])
    await row.getByRole('button', { name: 'Delete row', exact: true }).click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(row).toHaveCount(0)
    await expect(page.getByRole('row').filter({ hasText: 'Community club' })).toBeVisible()
    expect(deleted).toEqual([2])
    await page.reload()
    await expect(page.getByRole('row').filter({ hasText: 'Community club' })).toBeVisible()
    await expect(row).toHaveCount(0)
  })
}

test('a failed deletion keeps the organization visible and allows a deliberate retry', async ({ page }) => {
  const deleted = await mockOrganizations(page, { staff: true, failOnce: true })
  await page.goto('/organizations')
  const row = page.getByRole('row').filter({ hasText: 'Fixture QA club' })
  await row.getByRole('button', { name: 'Delete row', exact: true }).click()
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(page.getByText('Failed to delete organization', { exact: true })).toBeVisible()
  await expect(row).toBeVisible()
  expect(deleted).toEqual([2])
  await row.getByRole('button', { name: 'Delete row', exact: true }).click()
  await page.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(row).toHaveCount(0)
  expect(deleted).toEqual([2, 2])
})

test('a club manager without a platform role has no destructive organization controls', async ({ page }) => {
  const deleted = await mockOrganizations(page)
  await page.goto('/organizations')
  await expect(page.getByRole('row').filter({ hasText: 'Fixture QA club' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Delete row', exact: true })).toHaveCount(0)
  expect(deleted).toEqual([])
})
