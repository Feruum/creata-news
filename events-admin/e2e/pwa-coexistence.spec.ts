import { create, toBinary } from '@bufbuild/protobuf'
import { type BrowserContext, expect, type Page, test } from '@playwright/test'
import { GetMeResponseSchema, ListCategoriesResponseSchema } from '@studyverse/proto'

const origin = 'http://127.0.0.1:6872'
const identity = '11111111-1111-4111-8111-111111111111'

async function mockBackend(context: BrowserContext) {
  const state = { signedOut: false, offline: false }
  const cors = {
    'access-control-allow-origin': origin,
    'access-control-allow-credentials': 'true',
    'access-control-allow-methods': 'GET, POST, OPTIONS',
    'access-control-allow-headers':
      'content-type, connect-protocol-version, connect-timeout-ms, x-user-agent, traceparent, tracestate'
  }
  const session = () => ({
    id: 'test-session',
    active: true,
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
    identity: { id: identity, traits: { email: 'display@example.test', name: { first: 'Student' } } }
  })
  const whoami = async (route: Parameters<Parameters<BrowserContext['route']>[1]>[0]) =>
    route.fulfill({ status: state.signedOut ? 401 : 200, headers: cors, json: state.signedOut ? {} : session() })
  await context.route('**/sessions/whoami', whoami)
  await context.route('**/telemetry/v1/**', (route) => route.fulfill({ json: {} }))
  await context.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/auth/session') return whoami(route)
    if (path === '/api/schedule/my-du')
      return route.fulfill({
        json: {
          connection: {
            connected: true,
            studyYear: 2026,
            term: 1,
            firstWeekStart: '2026-09-01',
            lastSyncedAt: new Date().toISOString(),
            syncing: false,
            syncError: null
          },
          events: [
            {
              id: 'lesson',
              title: 'Shared saved lesson',
              start: new Date().toISOString(),
              end: new Date(Date.now() + 3_600_000).toISOString(),
              teacher: 'Teacher',
              lessonType: 'Lecture',
              location: 'C1.1.101',
              online: false
            }
          ]
        }
      })
    if (path === '/api/schedule/profile')
      return route.fulfill({ json: { groupName: 'SE-1', source: 'mydu', contributes: false, submission: null } })
    if (path === '/api/schedule/moodle') return route.fulfill({ json: { connection: null, events: [] } })
    if (path === '/api/schedule/groups') return route.fulfill({ json: [] })
    if (path.endsWith('/GetMe'))
      return route.fulfill({
        json: {
          user: {
            id: 1,
            username: 'student',
            email: 'display@example.test',
            firstName: 'Student',
            platformRole: 1
          }
        }
      })
    return route.fulfill({ json: {} })
  })
  await context.route('**/users.v1.UsersService/GetMe', (route) => {
    if (new URL(route.request().url()).pathname.startsWith('/api/')) return route.fallback()
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors })
    return route.fulfill({
      contentType: 'application/proto',
      headers: cors,
      body: Buffer.from(
        toBinary(
          GetMeResponseSchema,
          create(GetMeResponseSchema, {
            user: { id: 1, username: 'student', email: 'display@example.test' },
            permissions: { isAdmin: true, isStaff: true }
          })
        )
      )
    })
  })
  await context.route('**/events.v1.CategoriesService/ListCategories', (route) => {
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors })
    return route.fulfill({
      contentType: 'application/proto',
      headers: cors,
      body: Buffer.from(
        toBinary(
          ListCategoriesResponseSchema,
          create(ListCategoriesResponseSchema, {
            categories: [{ id: 1, name: 'Shared saved category', color: '#3B82F6' }],
            total: 1
          })
        )
      )
    })
  })
  await context.route('**/self-service/logout/browser**', (route) => {
    if (state.offline) return route.abort('internetdisconnected')
    state.signedOut = true
    return route.fulfill({ headers: cors, json: { logout_url: `${origin}/?signedOut=1` } })
  })
  await context.route('**/self-service/login/browser**', (route) =>
    route.fulfill({
      headers: cors,
      json: {
        id: 'synthetic-login-flow',
        ui: {
          action: `${origin}/__synthetic-login`,
          method: 'POST',
          nodes: [
            {
              group: 'oidc',
              attributes: { node_type: 'input', name: 'provider', type: 'submit', value: 'microsoft' }
            },
            {
              group: 'default',
              attributes: { node_type: 'input', name: 'csrf_token', type: 'hidden', value: 'synthetic-csrf' }
            }
          ]
        }
      }
    })
  )
  await context.route(`${origin}/__synthetic-login`, (route) => {
    state.signedOut = false
    return route.fulfill({ contentType: 'text/html', body: '<p>Signed in</p>' })
  })
  return state
}

async function controlled(page: Page, workerPath: string) {
  await page.waitForFunction((path) => navigator.serviceWorker.controller?.scriptURL.endsWith(path), workerPath)
}

async function warmBoth(context: BrowserContext, map: Page) {
  const state = await mockBackend(context)
  await map.goto('/')
  await expect(map.locator('g#FIRST_FLOOR')).toBeVisible()
  await controlled(map, '/serwist/sw.js')
  await map.getByRole('button', { name: 'Schedule', exact: true }).click()
  await expect(map.getByText('Shared saved lesson', { exact: true }).first()).toBeVisible()
  const admin = await context.newPage()
  await admin.goto('/administration/categories')
  await expect(admin.getByText('Shared saved category', { exact: true })).toBeVisible()
  await controlled(admin, '/administration/sw.js')
  await expect
    .poll(() =>
      admin.evaluate(async () => {
        const request = indexedDB.open('studyverse-admin-offline-v1:/administration/', 1)
        const db = await new Promise<IDBDatabase>((resolve) => {
          request.onsuccess = () => resolve(request.result)
        })
        const read = db.transaction('snapshots').objectStore('snapshots').get('active')
        const data = await new Promise<unknown>((resolve) => {
          read.onsuccess = () => resolve(read.result)
        })
        db.close()
        return JSON.stringify(data)?.includes('Shared saved category')
      })
    )
    .toBe(true)
  return { admin, state }
}

test('both real workers keep separate scopes and cold offline shells on one origin', async ({ context, page }) => {
  const { admin } = await warmBoth(context, page)
  const scopes = await page.evaluate(async () =>
    (await navigator.serviceWorker.getRegistrations()).map((entry) => entry.scope).sort()
  )
  expect(scopes).toEqual([`${origin}/`, `${origin}/administration/`])
  const cacheEntries = await page.evaluate(async () => {
    const entries = await Promise.all(
      (await caches.keys()).map(async (name) => ({
        name,
        paths: (await (await caches.open(name)).keys()).map((entry) => new URL(entry.url).pathname)
      }))
    )
    return entries
  })
  expect(cacheEntries.some((entry) => entry.name.startsWith('studyverse-map'))).toBe(true)
  expect(cacheEntries.some((entry) => entry.name.startsWith('studyverse-admin'))).toBe(true)
  expect(
    cacheEntries
      .filter((entry) => entry.name.startsWith('studyverse-map'))
      .flatMap((entry) => entry.paths)
      .some((path) => path.startsWith('/administration/'))
  ).toBe(false)
  await context.setOffline(true)
  await page.close()
  await admin.close()
  const coldMap = await context.newPage()
  await coldMap.goto('/')
  await expect(coldMap.locator('g#FIRST_FLOOR')).toBeVisible()
  await coldMap.getByRole('button', { name: 'Schedule', exact: true }).click()
  await expect(coldMap.getByText('Shared saved lesson', { exact: true }).first()).toBeVisible()
  const coldAdmin = await context.newPage()
  await coldAdmin.goto('/administration/categories')
  await expect(coldAdmin.getByText('Shared saved category', { exact: true })).toBeVisible()
  await coldMap.screenshot({ path: 'test-results/shared-origin-map-offline.png' })
  await coldAdmin.screenshot({ path: 'test-results/shared-origin-admin-offline.png' })
})

test('map logout revokes the actual admin private snapshot on the shared origin', async ({ context, page }) => {
  const { admin } = await warmBoth(context, page)
  await page.getByRole('button', { name: 'Close', exact: true }).first().click()
  await page.getByRole('button', { name: 'Open menu', exact: true }).click()
  await page.getByRole('button', { name: 'Sign out', exact: true }).click()
  await expect(admin.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await context.setOffline(true)
  await admin.reload()
  await expect(admin.getByText('Shared saved category', { exact: true })).toHaveCount(0)
})

test('explicit admin login clears offline map logout in the already open map tab', async ({ context, page }) => {
  const { admin, state } = await warmBoth(context, page)
  await page.getByRole('button', { name: 'Close', exact: true }).first().click()
  state.offline = true
  await context.setOffline(true)
  await expect(page.getByRole('status', { name: 'Offline access' })).toBeVisible()
  await page.getByRole('button', { name: 'Open menu', exact: true }).click()
  await page.getByRole('button', { name: 'Sign out', exact: true }).click()
  await expect(admin.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  expect(state.signedOut).toBe(false)
  state.offline = false
  await context.setOffline(false)
  await expect(page.getByRole('button', { name: 'Sign in with Microsoft', exact: true })).toBeVisible()
  await admin.getByRole('button', { name: 'Sign in with Microsoft', exact: true }).click()
  await admin.waitForURL(`${origin}/__synthetic-login`)
  await admin.goto('/administration/categories')
  await expect(admin.getByText('Shared saved category', { exact: true })).toBeVisible()
  await page.evaluate(() => window.dispatchEvent(new Event('online')))
  await expect(page.getByRole('button', { name: 'Sign out', exact: true })).toBeVisible()
})

test('updating either production worker preserves the other app caches and shows the reload action', async ({
  context,
  page,
  request
}) => {
  const { admin } = await warmBoth(context, page)
  await request.post('/__pwa-test/revision/admin')
  await admin.evaluate(async () => {
    await (await navigator.serviceWorker.getRegistration('/administration/'))?.update()
  })
  await expect(admin.getByText('An app update is ready.')).toBeVisible()
  await admin.getByRole('button', { name: 'Update', exact: true }).click()
  await expect(admin.getByText('Shared saved category', { exact: true })).toBeVisible()
  expect(await page.evaluate(async () => (await caches.keys()).some((name) => name.startsWith('studyverse-map')))).toBe(
    true
  )
  await request.post('/__pwa-test/revision/map')
  await page.evaluate(async () => {
    await (await navigator.serviceWorker.getRegistration('/'))?.update()
  })
  await page.getByRole('button', { name: 'Close', exact: true }).first().click()
  await page.getByRole('button', { name: 'Open menu', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Update available · reload' })).toBeVisible()
  await page.getByRole('button', { name: 'Update available · reload' }).click()
  await expect(page.locator('g#FIRST_FLOOR')).toBeVisible()
  expect(
    await admin.evaluate(async () => (await caches.keys()).some((name) => name.startsWith('studyverse-admin')))
  ).toBe(true)
  await context.setOffline(true)
  await admin.reload()
  await expect(admin.getByText('Shared saved category', { exact: true })).toBeVisible()
})
