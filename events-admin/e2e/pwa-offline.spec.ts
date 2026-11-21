import { create, toBinary } from '@bufbuild/protobuf'
import { type BrowserContext, expect, type Page, test } from '@playwright/test'
import { GetMeResponseSchema, ListCategoriesResponseSchema } from '@studyverse/proto'

const origin = 'http://127.0.0.1:6871'
const base = '/administration/'
const dbName = `studyverse-admin-offline-v1:${base}`

interface BackendState {
  account: string
  signedOut: boolean
  expiresAt: number
  writes: number
  holdNextSession: boolean
  releaseHeldSession: (() => Promise<void>) | null
  telemetry: { traces: string[]; metrics: string[]; logs: string[] }
}

async function mockBackend(context: BrowserContext) {
  const state: BackendState = {
    account: 'a',
    signedOut: false,
    expiresAt: Date.now() + 60 * 60 * 1000,
    writes: 0,
    holdNextSession: false,
    releaseHeldSession: null,
    telemetry: { traces: [], metrics: [], logs: [] }
  }
  const cors = {
    'access-control-allow-origin': origin,
    'access-control-allow-credentials': 'true',
    'access-control-allow-methods': 'GET, POST, OPTIONS',
    'access-control-allow-headers':
      'content-type, connect-protocol-version, connect-timeout-ms, x-user-agent, traceparent, tracestate, baggage'
  }
  await context.route('**/telemetry/v1/**', (route) => {
    const signal = new URL(route.request().url()).pathname.split('/').at(-1)
    if (signal === 'traces' || signal === 'metrics' || signal === 'logs') {
      state.telemetry[signal].push(route.request().postData() ?? '')
    }
    return route.fulfill({ json: {} })
  })
  await context.route('**/sessions/whoami', (route) => {
    if (state.holdNextSession) {
      state.holdNextSession = false
      state.releaseHeldSession = () => route.fulfill({ status: 401, headers: cors, json: { error: 'old session' } })
      return
    }
    return route.fulfill({
      status: state.signedOut ? 401 : 200,
      headers: cors,
      json: state.signedOut
        ? { error: 'unauthorized' }
        : {
            id: 'session-secret-never-persist',
            active: true,
            expires_at: new Date(state.expiresAt).toISOString(),
            identity: {
              id: `identity-${state.account}`,
              traits: { email: `${state.account}@example.test`, name: { first: `Account ${state.account}` } }
            }
          }
    })
  })
  await context.route('**/self-service/logout/browser**', (route) => {
    state.signedOut = true
    return route.fulfill({ headers: cors, json: { logout_url: `${origin}${base}auth/login` } })
  })
  await context.route('**/users.v1.UsersService/GetMe', (route) => {
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors })
    const message = create(GetMeResponseSchema, {
      user: {
        id: state.account === 'a' ? 1 : 2,
        username: `account-${state.account}`,
        email: `${state.account}@example.test`
      },
      permissions: { isAdmin: true, isStaff: true, managedClubIds: [4] }
    })
    return route.fulfill({
      contentType: 'application/proto',
      body: Buffer.from(toBinary(GetMeResponseSchema, message)),
      headers: cors
    })
  })
  await context.route('**/events.v1.CategoriesService/*', (route) => {
    const request = route.request()
    if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers: cors })
    if (!request.url().endsWith('/ListCategories')) {
      state.writes += 1
      return route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: '{"code":"unavailable"}',
        headers: cors
      })
    }
    const message = create(ListCategoriesResponseSchema, {
      categories: [{ id: state.account === 'a' ? 1 : 2, name: `Saved category ${state.account}`, color: '#3B82F6' }],
      total: 1
    })
    return route.fulfill({
      contentType: 'application/proto',
      body: Buffer.from(toBinary(ListCategoriesResponseSchema, message)),
      headers: cors
    })
  })
  return state
}

async function waitForShell(page: Page) {
  await page.evaluate(async () => {
    await navigator.serviceWorker.ready
  })
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true)
  await expect
    .poll(() =>
      page.evaluate(async (name) => {
        const db = await new Promise<IDBDatabase>((resolve, reject) => {
          const request = indexedDB.open(name, 1)
          request.onsuccess = () => resolve(request.result)
          request.onerror = () => reject(request.error)
        })
        const value = await new Promise<unknown>((resolve) => {
          const request = db.transaction('snapshots', 'readonly').objectStore('snapshots').get('active')
          request.onsuccess = () => resolve(request.result)
        })
        db.close()
        return (JSON.stringify(value) ?? '').includes('Saved category')
      }, dbName)
    )
    .toBe(true)
}

test('production manifest, worker and static caches stay under the admin scope', async ({ context, page }) => {
  await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  const manifest = await page.evaluate(async () => {
    const href = document.querySelector<HTMLLinkElement>('link[rel="manifest"]')!.href
    return (await fetch(href)).json() as Promise<{
      scope: string
      start_url: string
      id: string
      icons: { src: string }[]
    }>
  })
  expect(manifest.scope).toBe(base)
  expect(manifest.start_url).toBe(base)
  expect(new URL(manifest.id, `${origin}${base}manifest.webmanifest`).pathname).toBe(base)
  const scopes = await page.evaluate(async () =>
    (await navigator.serviceWorker.getRegistrations()).map((entry) => entry.scope)
  )
  expect(scopes).toEqual([`${origin}${base}`])
  const urls = await page.evaluate(async () => {
    const keys = await caches.keys()
    const all = await Promise.all(
      keys.map(async (key) => (await (await caches.open(key)).keys()).map((request) => request.url))
    )
    return { keys, urls: all.flat() }
  })
  expect(urls.keys.every((key) => key.startsWith('studyverse-admin'))).toBe(true)
  expect(urls.urls.every((url) => new URL(url).pathname.startsWith(base))).toBe(true)
  expect(urls.urls.some((url) => /\/(?:sessions|telemetry\/v1|oauth)|Service\//.test(new URL(url).pathname))).toBe(
    false
  )
})

test('cold offline launch keeps saved identity and pages while blocking writes without replay', async ({
  context,
  page
}) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  await context.setOffline(true)
  const cold = await context.newPage()
  await cold.goto(`${origin}${base}categories`)
  await expect(cold.getByText('Saved category a', { exact: true })).toBeVisible()
  await expect(cold.getByRole('status').filter({ hasText: 'Read-only' })).toBeVisible()
  await expect(cold.getByRole('button', { name: 'User menu for Account a' })).toBeVisible()
  await cold.getByRole('button', { name: 'Add category', exact: true }).click()
  await cold.getByRole('textbox', { name: 'Name', exact: true }).fill('Offline action')
  await cold.getByRole('button', { name: 'Create Category', exact: true }).click()
  await expect(cold.getByText('You are in read-only mode.', { exact: false })).toBeVisible()
  await expect(cold.getByRole('button', { name: 'Create Category', exact: true })).toBeEnabled()
  expect(backend.writes).toBe(0)
  await cold.getByRole('button', { name: 'Cancel', exact: true }).click()
  await context.setOffline(false)
  await expect(cold.getByRole('status').filter({ hasText: 'Read-only' })).toHaveCount(0)
  expect(backend.writes).toBe(0)
  await cold.getByRole('link', { name: 'Users', exact: true }).click()
  await context.setOffline(true)
  await cold.reload()
  await expect(cold.getByText(/no saved data|unavailable offline/).first()).toBeVisible()
})

test('confirmed sign-out removes private snapshots in all tabs', async ({ context, page }) => {
  await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  const other = await context.newPage()
  await other.goto(`${origin}${base}categories`)
  await expect(other.getByText('Saved category a', { exact: true })).toBeVisible()
  await context.setOffline(true)
  await page.getByRole('button', { name: 'User menu for Account a' }).click()
  await page.getByRole('menuitem', { name: 'Log out' }).click()
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await expect(other.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await other.reload()
  await expect(other.getByText('Saved category a', { exact: true })).toHaveCount(0)
  await context.setOffline(false)
  // The server cookie is still live because offline logout could not reach Kratos.
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
})

test('a late failed session check cannot revoke a newer verified account', async ({ context, page }) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  backend.holdNextSession = true
  await page.evaluate(() => window.dispatchEvent(new Event('online')))
  await expect.poll(() => !!backend.releaseHeldSession).toBe(true)
  backend.account = 'b'
  await page.evaluate(() => window.dispatchEvent(new Event('online')))
  await expect(page.getByRole('button', { name: 'User menu for Account b' })).toBeVisible()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
  await backend.releaseHeldSession?.()
  await expect(page.getByRole('button', { name: 'User menu for Account b' })).toBeVisible()
  await context.setOffline(true)
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
})

test('a delayed initial 401 cannot delete an account verified by another tab', async ({ context, page }) => {
  const backend = await mockBackend(context)
  backend.holdNextSession = true
  await page.goto('categories')
  await expect.poll(() => !!backend.releaseHeldSession).toBe(true)
  backend.account = 'b'
  const newer = await context.newPage()
  await newer.goto(`${origin}${base}categories`)
  await expect(newer.getByText('Saved category b', { exact: true })).toBeVisible()
  await waitForShell(newer)
  await backend.releaseHeldSession?.()
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await context.setOffline(true)
  await newer.reload()
  await expect(newer.getByText('Saved category b', { exact: true })).toBeVisible()
})

test('cross-app logout clears matching private data and a delayed old-account message preserves a newer account', async ({
  context,
  page
}) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  backend.account = 'b'
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
  await page.evaluate(() => {
    const channel = new BroadcastChannel('studyverse-accounts')
    channel.postMessage({
      kind: 'account-revoked',
      source: 'map',
      reason: 'logout',
      previousIdentityId: 'identity-a',
      nonce: 'late-a'
    })
    channel.close()
  })
  await context.setOffline(true)
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
  await page.evaluate(() => {
    const channel = new BroadcastChannel('studyverse-accounts')
    channel.postMessage({
      kind: 'account-revoked',
      source: 'map',
      reason: 'logout',
      previousIdentityId: 'identity-b',
      nonce: 'current-b'
    })
    channel.close()
  })
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toHaveCount(0)
})

test('a verified account switch replaces rather than mixes snapshots', async ({ context, page }) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  backend.account = 'b'
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
  await context.setOffline(true)
  await page.reload()
  await expect(page.getByText('Saved category b', { exact: true })).toBeVisible()
  await expect(page.getByText('Saved category a', { exact: true })).toHaveCount(0)
})

test('expired snapshots cannot restore a protected view on cold offline launch', async ({ context, page }) => {
  await mockBackend(context)
  await page.clock.install({ time: Date.now() })
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  await context.setOffline(true)
  await page.clock.fastForward(60 * 60 * 1000 + 1)
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible({ timeout: 10_000 })
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await expect(page.getByText('Saved category a', { exact: true })).toHaveCount(0)
})

test('blocked storage permits online use and readable quota-limited storage preserves offline logout', async ({
  context,
  page
}) => {
  const backend = await mockBackend(context)
  await page.addInitScript(() => {
    Object.defineProperty(window, 'localStorage', {
      get: () => {
        throw new DOMException('Blocked', 'SecurityError')
      }
    })
    Object.defineProperty(window, 'sessionStorage', {
      get: () => {
        throw new DOMException('Blocked', 'SecurityError')
      }
    })
    indexedDB.open = () => {
      throw new DOMException('Blocked', 'SecurityError')
    }
  })
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await expect(page.getByText(/Offline storage is unavailable/)).toBeVisible()
  await page.getByRole('button', { name: 'Theme: Dark' }).click()
  await page.getByRole('menuitemradio', { name: 'Light', exact: true }).click()
  await expect(page.locator('html')).toHaveClass(/light/)

  const quotaPage = await context.newPage()
  await quotaPage.addInitScript(() => {
    Storage.prototype.setItem = () => {
      throw new DOMException('Full', 'QuotaExceededError')
    }
  })
  await quotaPage.goto(`${origin}${base}categories`)
  await expect(quotaPage.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(quotaPage)
  await context.setOffline(true)
  await quotaPage.getByRole('button', { name: 'User menu for Account a' }).click()
  await quotaPage.getByRole('menuitem', { name: 'Log out' }).click()
  await expect(quotaPage.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  expect(await quotaPage.evaluate(() => localStorage.getItem('studyverse-logout-pending'))).toBeNull()
  const checked = quotaPage.waitForResponse((response) => new URL(response.url()).pathname === '/sessions/whoami')
  await context.setOffline(false)
  expect((await checked).status()).toBe(200)
  await expect(quotaPage.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  expect(backend.signedOut).toBe(false)
})

test('offline logout survives a cold admin page when localStorage writes are blocked', async ({ context, page }) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  const blockWrites = () => {
    Storage.prototype.setItem = () => {
      throw new DOMException('Full', 'QuotaExceededError')
    }
  }
  await context.addInitScript(blockWrites)
  await page.evaluate(blockWrites)
  await context.setOffline(true)
  await page.getByRole('button', { name: 'User menu for Account a' }).click()
  await page.getByRole('menuitem', { name: 'Log out' }).click()
  await expect(page.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await page.close()
  const cold = await context.newPage()
  await cold.goto(`${origin}${base}auth/login`)
  await expect(cold.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  const checked = cold.waitForResponse((response) => new URL(response.url()).pathname === '/sessions/whoami')
  await context.setOffline(false)
  expect((await checked).status()).toBe(200)
  await expect(cold.getByRole('heading', { name: 'Welcome Back' })).toBeVisible()
  await expect(cold.getByText('Saved category a', { exact: true })).toHaveCount(0)
  expect(backend.signedOut).toBe(false)
})

test('sidebar theme persists and System subscribes to OS appearance changes', async ({ context, page }) => {
  await mockBackend(context)
  await page.goto('categories')
  await page.getByRole('button', { name: 'Theme: Dark' }).focus()
  await page.getByRole('button', { name: 'Theme: Dark' }).press('Enter')
  await page.getByRole('menuitemradio', { name: 'Light', exact: true }).press('Enter')
  await page.reload()
  await expect(page.locator('html')).toHaveClass(/light/)
  await page.getByRole('button', { name: 'Theme: Light' }).click()
  await page.getByRole('menuitemradio', { name: 'System', exact: true }).click()
  await page.emulateMedia({ colorScheme: 'dark' })
  await expect(page.locator('html')).toHaveClass(/dark/)
  await page.emulateMedia({ colorScheme: 'light' })
  await expect(page.locator('html')).toHaveClass(/light/)
  expect(await page.evaluate(() => localStorage.getItem('vite-ui-theme'))).toBe('system')
})

test('sidebar installation opens the browser prompt and iOS receives Safari instructions', async ({
  context,
  page,
  browser
}) => {
  await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await page.evaluate(() => {
    const target = window as unknown as { installPrompts: number }
    target.installPrompts = 0
    const event = new Event('beforeinstallprompt', { cancelable: true })
    Object.defineProperties(event, {
      prompt: {
        value: async () => {
          target.installPrompts += 1
        }
      },
      userChoice: { value: Promise.resolve({ outcome: 'accepted' }) }
    })
    window.dispatchEvent(event)
  })
  await page.getByRole('button', { name: 'Install AITU EMS', exact: true }).click()
  expect(await page.evaluate(() => (window as unknown as { installPrompts: number }).installPrompts)).toBe(1)
  const ios = await browser.newContext({
    userAgent:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1'
  })
  await mockBackend(ios)
  const iosPage = await ios.newPage()
  await iosPage.goto(`${origin}${base}categories`)
  await iosPage.getByRole('button', { name: 'Install AITU EMS', exact: true }).click()
  await expect(iosPage.getByRole('dialog')).toContainText('Tap Share, then Add to Home Screen.')
  await ios.close()
})

test('a real waiting worker offers an explicit update and takes control only after acceptance', async ({
  context,
  page
}) => {
  await mockBackend(context)
  await page.goto('categories')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
  await waitForShell(page)
  await context.route('**/sw.js?test-update=2', async (route) => {
    const original = await route.fetch()
    await route.fulfill({ response: original, body: `${await original.text()}\n// test update version two\n` })
  })
  await page.evaluate(async () => {
    await navigator.serviceWorker.register('/administration/sw.js?test-update=2', { scope: '/administration/' })
  })
  await expect(page.getByRole('button', { name: 'Update', exact: true })).toBeVisible()
  expect(await page.evaluate(() => navigator.serviceWorker.controller?.scriptURL)).not.toContain('test-update=2')
  const reload = page.waitForEvent('load')
  await page.getByRole('button', { name: 'Update', exact: true }).click()
  await reload
  await expect.poll(() => page.evaluate(() => navigator.serviceWorker.controller?.scriptURL)).toContain('test-update=2')
  await expect(page.getByText('Saved category a', { exact: true })).toBeVisible()
})

test('the real browser SDK emits all three OTLP signals and a bounded theme counter', async ({ context, page }) => {
  const backend = await mockBackend(context)
  await page.goto('categories')
  await page.getByRole('button', { name: 'Theme: Dark' }).click()
  await page.getByRole('menuitemradio', { name: 'Light', exact: true }).click()
  await expect.poll(() => backend.telemetry.traces.length, { timeout: 20_000 }).toBeGreaterThan(0)
  await page.evaluate(() => window.dispatchEvent(new ErrorEvent('error', { error: new Error('private-note-432') })))
  await expect.poll(() => backend.telemetry.logs.length, { timeout: 20_000 }).toBeGreaterThan(0)
  await expect.poll(() => backend.telemetry.metrics.join(''), { timeout: 25_000 }).toContain('theme_change')
  expect(backend.telemetry.metrics.join('')).toContain('studyverse.ui.events')
  const combined = Object.values(backend.telemetry).flat().join('')
  expect(combined).not.toContain('a@example.test')
  expect(combined).not.toContain('session-secret-never-persist')
  expect(combined).not.toContain('private-note-432')
})
