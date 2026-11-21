import { expect, type Page, test } from '@playwright/test'

const ownerInfo = {
  claimed: true,
  telegram_linked: false,
  telegram_configured: false,
  gitlab_configured: false,
  chatgpt_connected: false,
  ready: false,
  dashboard_url: 'https://auth.webspace.cat/hermes/'
}

declare global {
  interface Window {
    hermesRequests: { method: string; credentials?: RequestCredentials; cache?: RequestCache; header: string | null }[]
  }
}

async function mockIdentity(page: Page, origin: string, isAdmin = false, pendingEmail = false) {
  const headers = {
    'access-control-allow-origin': origin,
    'access-control-allow-credentials': 'true',
    'access-control-allow-headers': 'Content-Type, X-Studyverse-Hermes'
  }
  await page.route('**/*', (route) => {
    if (new URL(route.request().url()).origin !== origin) return route.abort()
    return route.continue()
  })
  await page.route('**/sessions/whoami', (route) =>
    route.fulfill({
      headers,
      json: {
        id: 'hermes-owner-session',
        active: true,
        expires_at: '2099-01-01T00:00:00.000Z',
        identity: {
          id: 'hermes-owner',
          traits: { email: 'display@example.test', name: 'Owner' },
          verifiable_addresses: pendingEmail
            ? [{ value: 'display@example.test', via: 'email', verified: false, status: 'sent' }]
            : []
        }
      }
    })
  )
  await page.route('**/users.v1.UsersService/GetMe', (route) =>
    route.fulfill({
      headers,
      json: {
        user: { id: 1, username: 'owner', email: 'mutable-display@example.test' },
        permissions: { isAdmin, isStaff: false, managedClubIds: [4] }
      }
    })
  )
  await page.route('**/self-service/logout/browser**', (route) => route.fulfill({ status: 401, json: {}, headers }))
  await page.route('**/search.v1.SearchService/GlobalSearch', (route) =>
    route.fulfill({ json: { results: [], totalHits: 0 }, headers })
  )
  await page.route('**/ListOrganizations', (route) => route.fulfill({ json: { organizations: [], total: 0 }, headers }))
  await page.route('**/ListCategories', (route) => route.fulfill({ json: { categories: [], total: 0 }, headers }))
  await page.route('**/ListRooms', (route) => route.fulfill({ json: { rooms: [], total: 0 }, headers }))
  await page.addInitScript(() => {
    window.hermesRequests = []
    const nativeFetch = window.fetch
    window.fetch = (input, init) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      if (new URL(url, window.location.origin).pathname.startsWith('/admin/hermes/')) {
        window.hermesRequests.push({
          method: init?.method ?? 'GET',
          credentials: init?.credentials,
          cache: init?.cache,
          header: new Headers(init?.headers).get('X-Studyverse-Hermes')
        })
      }
      return nativeFetch(input, init)
    }
  })
  return headers
}

test('configured owner with an unverified email can open Hermes directly', async ({ page, baseURL }) => {
  const headers = await mockIdentity(page, new URL(baseURL ?? '').origin, false, true)
  await page.route('**/admin/hermes/info', (route) => route.fulfill({ headers, json: ownerInfo }))
  const writes: string[] = []
  await page.route('**/admin/hermes/settings', (route) => {
    writes.push(route.request().method())
    return route.abort()
  })
  await page.goto('/categories')
  const nav = page.getByRole('link', { name: 'Hermes', exact: true })
  await expect(nav).toBeVisible()
  await nav.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('heading', { name: 'Hermes', exact: true })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Verify email' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Claim Hermes' })).toHaveCount(0)
  const dashboard = page.getByRole('link', { name: 'Open Hermes', exact: true })
  await expect(dashboard).toHaveAttribute('href', ownerInfo.dashboard_url)
  await expect(dashboard).toHaveAttribute('target', '_blank')
  await expect(dashboard).toHaveAttribute('rel', 'noopener noreferrer')
  expect(writes).toEqual([])
  await expect(page.getByText(/choose Open menu and link Telegram in your account card/)).toBeVisible()
  await expect(page.getByRole('link', { name: 'Open AITU Map', exact: true })).toHaveAttribute(
    'href',
    'https://map.webspace.cat/'
  )
  for (const request of await page.evaluate(() => window.hermesRequests)) {
    expect(request).toMatchObject({ credentials: 'include', cache: 'no-store', header: '1' })
  }
})

for (const status of [403, 404, 503]) {
  test(`Hermes navigation is absent when owner authorization returns ${status}, even for an admin`, async ({
    page,
    baseURL
  }) => {
    const headers = await mockIdentity(page, new URL(baseURL ?? '').origin, true, true)
    await page.route('**/admin/hermes/info', (route) =>
      route.fulfill({ status, json: { error: 'Unavailable' }, headers })
    )
    await page.goto('/categories')
    await expect(page.getByRole('heading', { name: 'Categories', exact: true })).toBeVisible()
    await expect.poll(() => page.evaluate(() => window.hermesRequests.length)).toBeGreaterThan(0)
    await expect(page.getByRole('link', { name: 'Hermes', exact: true })).toHaveCount(0)
    await page.goto('/hermes')
    await expect(page.getByRole('alert')).toContainText(status === 403 ? 'only to its owner' : 'unavailable')
    await expect(page.getByRole('link', { name: 'Verify email' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Claim Hermes', exact: true })).toHaveCount(0)
    await expect(page.getByRole('link', { name: 'Open Hermes', exact: true })).toHaveCount(0)
  })
}

test('the independent Kratos account verification page can still complete a code flow', async ({ page, baseURL }) => {
  const origin = new URL(baseURL ?? '').origin
  const headers = await mockIdentity(page, origin, true, true)
  const email = 'display@example.test'
  const id = '76e54190-5788-44f5-97d9-ae7643d02817'
  let state = 'choose_method'
  let invalidCode = false
  let requests = 0
  const redirect = { status: 303, headers: { location: origin + '/auth/verification?flow=' + id } }
  await page.route('**/self-service/verification/browser*', (route) => {
    expect(new URL(route.request().url()).searchParams.has('return_to')).toBe(false)
    return route.fulfill(redirect)
  })
  await page.route('**/self-service/verification/flows?*', (route) => {
    const input = (name: string, type: string, value: string) => ({
      attributes: { node_type: 'input', name, type, value },
      messages: []
    })
    return route.fulfill({
      headers,
      json: {
        id,
        type: 'browser',
        state,
        return_to: origin + '/',
        ui: {
          method: 'POST',
          action: new URL('/self-service/verification?flow=' + id, route.request().url()).href,
          messages: invalidCode ? [{ id: 1, type: 'error', text: 'That code is invalid.' }] : [],
          nodes: [
            input('csrf_token', 'hidden', 'test-csrf-' + requests),
            input('method', 'submit', 'code'),
            ...(state === 'sent_email'
              ? [input('code', 'text', ''), input('email', 'submit', email)]
              : [input('email', 'email', '')])
          ]
        }
      }
    })
  })
  await page.route('**/self-service/verification?*', (route) => {
    expect(route.request().method()).toBe('POST')
    const body = new URLSearchParams(route.request().postData() ?? '')
    expect(body.get('csrf_token')).toBe('test-csrf-' + requests++)
    expect(body.get('method')).toBe('code')
    if (body.has('email')) {
      expect(body.get('email')).toBe(email)
      expect(body.has('code')).toBe(false)
      state = 'sent_email'
    } else {
      invalidCode = body.get('code') !== '000042'
      if (!invalidCode) state = 'passed_challenge'
    }
    return route.fulfill(redirect)
  })
  await page.goto('/auth/verification')
  await page.getByRole('link', { name: 'Start verification' }).click()
  await expect(page.getByLabel('Email address')).toHaveValue(email)
  await page.getByRole('button', { name: 'Send verification code' }).click()
  await page.getByRole('button', { name: 'Send a new code' }).click()
  await page.getByLabel('Verification code').fill('111111')
  await page.getByRole('button', { name: 'Verify code' }).click()
  await expect(page.getByRole('alert')).toContainText('That code is invalid')
  await page.getByLabel('Verification code').fill('000042')
  await page.getByRole('button', { name: 'Verify code' }).click()
  await expect(page.getByRole('link', { name: 'Return to administration' })).toBeVisible()
  expect(requests).toBe(4)
})

test('an expired verification flow offers a fresh browser request', async ({ page, baseURL }) => {
  const origin = new URL(baseURL ?? '').origin
  const headers = await mockIdentity(page, origin)
  await page.route('**/self-service/verification/flows?*', (route) => route.fulfill({ headers, status: 410, json: {} }))
  await page.goto('/auth/verification?flow=76e54190-5788-44f5-97d9-ae7643d02817')
  await expect(page.getByRole('link', { name: 'Start verification' })).toHaveAttribute(
    'href',
    /\/self-service\/verification\/browser/
  )
  await expect(page.getByLabel('Verification code')).toHaveCount(0)
})

test('malformed dashboard URLs never become outbound links', async ({ page, baseURL }) => {
  const headers = await mockIdentity(page, new URL(baseURL ?? '').origin)
  await page.route('**/admin/hermes/info', (route) =>
    route.fulfill({
      headers,
      json: { ...ownerInfo, claimed: true, dashboard_url: 'https://evil.example/hermes/' }
    })
  )
  await page.goto('/hermes')
  await expect(page.getByRole('alert')).toContainText('unavailable')
  await expect(page.getByRole('link', { name: 'Open Hermes', exact: true })).toHaveCount(0)
  await expect(page.locator('[data-sidebar="sidebar"]').getByRole('link', { name: 'Hermes', exact: true })).toHaveCount(
    0
  )
})

test('an unavailable owner check can be retried without retaining private response fields', async ({
  page,
  baseURL
}) => {
  const headers = await mockIdentity(page, new URL(baseURL ?? '').origin)
  let unavailable = true
  await page.route('**/admin/hermes/info', (route) =>
    unavailable
      ? route.fulfill({ status: 503, headers, json: { error: 'private-server-diagnostic' } })
      : route.fulfill({
          headers,
          json: {
            ...ownerInfo,
            bot_token: 'synthetic-private-token',
            recent_incidents: [{ summary: 'private-diagnosis' }]
          }
        })
  )
  await page.goto('/hermes')
  await expect(page.getByRole('alert')).toContainText('unavailable')
  unavailable = false
  await page.getByRole('button', { name: 'Try again', exact: true }).click()
  await expect(page.getByRole('link', { name: 'Open Hermes', exact: true })).toBeVisible()
  const html = await page.content()
  for (const value of ['synthetic-private-token', 'private-diagnosis', 'private-server-diagnostic'])
    expect(html).not.toContain(value)
  const storage = await page.evaluate(() =>
    JSON.stringify({ local: { ...localStorage }, session: { ...sessionStorage } })
  )
  expect(storage).not.toContain('hermes')
})

test('owner launcher remains usable on a narrow screen and disappears offline', async ({ page, baseURL }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const headers = await mockIdentity(page, new URL(baseURL ?? '').origin)
  await page.route('**/admin/hermes/info', (route) => route.fulfill({ headers, json: { ...ownerInfo, claimed: true } }))
  await page.goto('/hermes')
  await expect(page.getByRole('link', { name: 'Open Hermes', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.evaluate(() => window.dispatchEvent(new Event('offline')))
  await expect(page.getByRole('link', { name: 'Open Hermes', exact: true })).toHaveCount(0)
})
