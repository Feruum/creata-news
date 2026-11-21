import { expect, type Page, test } from '@playwright/test'

const endpoint = '/admin/signoz/info'
const connection = {
  configured: true,
  url: 'https://auth.webspace.cat',
  login: { email: 'synthetic-signoz@example.test', password: 'synthetic-signoz-password' }
}
const cors = {
  'access-control-allow-origin': 'http://localhost:6868',
  'access-control-allow-credentials': 'true',
  'access-control-allow-headers': 'X-Studyverse-SigNoz'
}

declare global {
  interface Window {
    signozRequests: {
      url: string
      method: string
      credentials: RequestCredentials | undefined
      cache: RequestCache | undefined
      header: string | null
      signal: AbortSignal | null | undefined
    }[]
    resolveSignoz?: () => void
    signozResolved?: boolean
  }
}

async function mockAuth(
  page: Page,
  options: { isAdmin?: boolean; isStaff?: boolean; identityId?: string; sessionId?: string; expiresAt?: string } = {}
) {
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      json: {
        id: options.sessionId ?? 'test-session',
        active: true,
        expires_at: options.expiresAt ?? '2099-01-01T00:00:00.000Z',
        identity: {
          id: options.identityId ?? 'admin-identity',
          traits: { email: 'admin@example.test', name: 'Admin' }
        }
      },
      headers: cors
    })
  )
  await page.route('**/users.v1.UsersService/GetMe', (route) =>
    route.fulfill({
      json: {
        user: { id: 1, username: 'reviewer', email: 'admin@example.test' },
        permissions: { isAdmin: options.isAdmin ?? true, isStaff: options.isStaff ?? false, managedClubIds: [4] }
      },
      headers: cors
    })
  )
  await page.route('**/self-service/logout/browser**', (route) =>
    route.fulfill({ status: 401, json: {}, headers: cors })
  )
  await page.route('**/search.v1.SearchService/GlobalSearch', (route) =>
    route.fulfill({ json: { results: [], totalHits: 0 }, headers: cors })
  )
  await page.route('**/ListOrganizations', (route) =>
    route.fulfill({ json: { organizations: [], total: 0 }, headers: cors })
  )
  await page.route('**/ListCategories', (route) => route.fulfill({ json: { categories: [], total: 0 }, headers: cors }))
  await page.route('**/ListRooms', (route) => route.fulfill({ json: { rooms: [], total: 0 }, headers: cors }))
}

async function mockInfo(page: Page, body: unknown = connection, status = 200) {
  await page.route(`**${endpoint}`, (route) =>
    route.fulfill({ status, json: body, headers: { ...cors, 'cache-control': 'no-store' } })
  )
}

async function openInfo(page: Page) {
  await page.getByRole('button', { name: 'SigNoz', exact: true }).click()
  return page.getByRole('dialog', { name: 'SigNoz', exact: true })
}

// Intentionally ignore abort so these races also prove stale response invalidation.
async function delayFirstInfo(page: Page) {
  await page.addInitScript((body) => {
    const nativeFetch = window.fetch
    let delayed = false
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (new URL(url, window.location.origin).pathname === '/admin/signoz/info' && !delayed) {
        delayed = true
        window.signozRequests.push({
          url,
          method: init?.method ?? 'GET',
          credentials: init?.credentials,
          cache: init?.cache,
          header: new Headers(init?.headers).get('X-Studyverse-SigNoz'),
          signal: init?.signal
        })
        return new Promise<Response>((resolve) => {
          window.resolveSignoz = () => {
            resolve(
              new Response(JSON.stringify(body), { status: 200, headers: { 'content-type': 'application/json' } })
            )
            window.signozResolved = true
          }
        })
      }
      return nativeFetch(input, init)
    }
  }, connection)
}

async function resolveDelayedInfo(page: Page) {
  await page.evaluate(async () => {
    window.resolveSignoz?.()
    await new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  })
  expect(await page.evaluate(() => window.signozResolved)).toBe(true)
}

async function assertNoCredentials(page: Page) {
  await expect(page.getByLabel('SigNoz email', { exact: true })).toHaveCount(0)
  await expect(page.getByLabel('SigNoz password', { exact: true })).toHaveCount(0)
  expect(await page.content()).not.toContain(connection.login.email)
  expect(await page.content()).not.toContain(connection.login.password)
}

test.beforeEach(async ({ page }) => {
  await page.route('**/*', (route) => {
    if (new URL(route.request().url()).origin !== 'http://localhost:6868') return route.abort()
    return route.continue()
  })
  await page.addInitScript(() => {
    window.signozRequests = []
    const nativeFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (new URL(url, window.location.origin).pathname === '/admin/signoz/info') {
        window.signozRequests.push({
          url,
          method: init?.method ?? 'GET',
          credentials: init?.credentials,
          cache: init?.cache,
          header: new Headers(init?.headers).get('X-Studyverse-SigNoz'),
          signal: init?.signal
        })
      }
      return nativeFetch(input, init)
    }
  })
  await mockAuth(page)
})

test('administrator opens SigNoz with keyboard and fetches access only on demand', async ({ page }) => {
  await mockInfo(page)
  await page.goto('/categories')
  const trigger = page.getByRole('button', { name: 'SigNoz', exact: true })
  await expect(trigger).toBeVisible()
  expect(await page.evaluate(() => window.signozRequests.length)).toBe(0)
  await trigger.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'SigNoz', exact: true })
  const link = dialog.getByRole('link', { name: 'Open SigNoz', exact: true })
  await expect(link).toHaveAttribute('href', connection.url)
  await expect(link).toHaveAttribute('target', '_blank')
  await expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  const email = dialog.getByLabel('SigNoz email', { exact: true })
  const password = dialog.getByLabel('SigNoz password', { exact: true })
  await expect(email).toHaveAttribute('type', 'password')
  await expect(password).toHaveAttribute('type', 'password')
  await expect(email).toHaveValue(connection.login.email)
  await expect(password).toHaveValue(connection.login.password)
  await dialog.getByRole('button', { name: 'Show login details', exact: true }).click()
  await expect(email).toHaveAttribute('type', 'text')
  await expect(password).toHaveAttribute('type', 'text')
  await dialog.getByRole('button', { name: 'Hide login details', exact: true }).click()
  await expect(password).toHaveAttribute('type', 'password')
  expect(
    await page.evaluate(() =>
      window.signozRequests.map(({ url, method, credentials, cache, header }) => ({
        path: new URL(url).pathname,
        method,
        credentials,
        cache,
        header
      }))
    )
  ).toEqual([{ path: endpoint, method: 'GET', credentials: 'include', cache: 'no-store', header: '1' }])
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  await expect(trigger).toBeFocused()
  await assertNoCredentials(page)
})

for (const permissions of [
  { isAdmin: false, isStaff: true },
  { isAdmin: false, isStaff: false }
]) {
  test(`SigNoz is absent for ${permissions.isStaff ? 'staff' : 'organization managers'}`, async ({ page }) => {
    await mockAuth(page, permissions)
    await mockInfo(page)
    await page.goto('/categories')
    await expect(page.getByRole('heading', { name: 'Categories', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'SigNoz', exact: true })).toHaveCount(0)
    expect(await page.evaluate(() => window.signozRequests.length)).toBe(0)
  })
}

test('unconfigured SigNoz shows an unavailable state without an outbound link', async ({ page }) => {
  await mockInfo(page, { configured: false })
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByText('SigNoz is unavailable.', { exact: true })).toBeVisible()
  await expect(dialog.getByRole('link')).toHaveCount(0)
  await assertNoCredentials(page)
})

test('operator-managed account opens the native UI without managed credentials', async ({ page }) => {
  await mockInfo(page, { configured: true, url: connection.url })
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByRole('link', { name: 'Open SigNoz', exact: true })).toHaveAttribute('href', connection.url)
  await expect(dialog.getByText('Sign in with your SigNoz account.', { exact: true })).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Show login details', exact: true })).toHaveCount(0)
  await assertNoCredentials(page)
})

test('forbidden response shows a generic error without returned credentials', async ({ page }) => {
  await mockInfo(page, { error: 'Forbidden', ...connection }, 403)
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByRole('alert')).toHaveText('Unable to load SigNoz access. Close this dialog and try again.')
  await expect(dialog.getByRole('link')).toHaveCount(0)
  await assertNoCredentials(page)
})

const malformed = [
  { configured: 'true', url: connection.url },
  { configured: false, ...{ url: connection.url, login: connection.login } },
  { configured: true, url: connection.url, login: { email: connection.login.email } },
  { configured: true, url: 'http://auth.webspace.cat', login: connection.login },
  { configured: true, url: 'https://user:pass@auth.webspace.cat', login: connection.login },
  { configured: true, url: 'https://auth.webspace.cat/path', login: connection.login },
  { configured: true, url: 'https://auth.webspace.cat?next=1', login: connection.login },
  { configured: true, url: 'https://auth.webspace.cat#fragment', login: connection.login }
]
for (const [index, body] of malformed.entries()) {
  test(`malformed response ${index + 1} fails closed without credentials`, async ({ page }) => {
    await mockInfo(page, body)
    await page.goto('/categories')
    const dialog = await openInfo(page)
    await expect(dialog.getByRole('alert')).toHaveText('Unable to load SigNoz access. Close this dialog and try again.')
    await expect(dialog.getByRole('link')).toHaveCount(0)
    await assertNoCredentials(page)
  })
}

test('closing clears login details and reopening starts a fresh masked request', async ({ page }) => {
  await mockInfo(page)
  await page.goto('/categories')
  let dialog = await openInfo(page)
  await dialog.getByRole('button', { name: 'Show login details', exact: true }).click()
  await expect(dialog.getByLabel('SigNoz password')).toHaveAttribute('type', 'text')
  await dialog.getByRole('button', { name: 'Close', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await assertNoCredentials(page)
  dialog = await openInfo(page)
  await expect(dialog.getByLabel('SigNoz password')).toHaveAttribute('type', 'password')
  expect(await page.evaluate(() => window.signozRequests.length)).toBe(2)
})

test('close aborts and ignores a delayed response even when fetch ignores abort', async ({ page }) => {
  await delayFirstInfo(page)
  await mockInfo(page, { configured: false })
  await page.goto('/categories')
  let dialog = await openInfo(page)
  await expect(dialog.getByRole('status')).toHaveText('Loading SigNoz access...')
  await dialog.getByRole('button', { name: 'Close', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  expect(await page.evaluate(() => window.signozRequests[0]?.signal?.aborted)).toBe(true)
  dialog = await openInfo(page)
  await expect(dialog.getByText('SigNoz is unavailable.', { exact: true })).toBeVisible()
  await resolveDelayedInfo(page)
  await expect(dialog.getByRole('link')).toHaveCount(0)
  await assertNoCredentials(page)
})

test('logout discards and ignores a delayed response', async ({ page }) => {
  await delayFirstInfo(page)
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByRole('status')).toBeVisible()
  // A sign-out in another administration tab invalidates the current session.
  await page.evaluate(async () => {
    const path = '/src/shared/lib/offline-runtime.ts'
    const { clearPrivateSnapshots } = await import(path)
    await clearPrivateSnapshots('logout')
  })
  await expect(page.getByRole('button', { name: 'SigNoz', exact: true })).toHaveCount(0)
  await expect(dialog).toHaveCount(0)
  expect(await page.evaluate(() => window.signozRequests[0]?.signal?.aborted)).toBe(true)
  await resolveDelayedInfo(page)
  await assertNoCredentials(page)
  await expect(page.getByRole('dialog', { name: 'SigNoz', exact: true })).toHaveCount(0)
})

test('account change discards access and ignores the previous account response', async ({ page }) => {
  await delayFirstInfo(page)
  await mockInfo(page, { configured: false })
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByRole('status')).toBeVisible()
  await mockAuth(page, { identityId: 'different-admin', sessionId: 'different-session' })
  await page.evaluate(() => window.dispatchEvent(new Event('online')))
  await expect(dialog).toHaveCount(0)
  const freshDialog = await openInfo(page)
  await expect(freshDialog.getByText('SigNoz is unavailable.', { exact: true })).toBeVisible()
  await resolveDelayedInfo(page)
  await assertNoCredentials(page)
  await expect(freshDialog.getByRole('link')).toHaveCount(0)
})

test('going offline clears revealed login and hides the entry point', async ({ page, context }) => {
  await mockInfo(page)
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await dialog.getByRole('button', { name: 'Show login details', exact: true }).click()
  await expect(dialog.getByLabel('SigNoz password')).toHaveAttribute('type', 'text')
  await context.setOffline(true)
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'SigNoz', exact: true })).toHaveCount(0)
  await assertNoCredentials(page)
})

test('session expiry clears the dialog and removes SigNoz access', async ({ page }) => {
  await page.clock.install()
  const now = await page.evaluate(() => Date.now())
  await mockAuth(page, { expiresAt: new Date(now + 60_000).toISOString() })
  await mockInfo(page)
  await page.goto('/categories')
  const dialog = await openInfo(page)
  await expect(dialog.getByLabel('SigNoz password')).toHaveValue(connection.login.password)
  await page.clock.fastForward(60_001)
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'SigNoz', exact: true })).toHaveCount(0)
  await assertNoCredentials(page)
})
