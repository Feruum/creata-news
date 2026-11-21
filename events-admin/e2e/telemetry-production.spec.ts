import { mkdir, writeFile } from 'node:fs/promises'
import { create, toBinary } from '@bufbuild/protobuf'
import { type BrowserContext, expect, test } from '@playwright/test'
import { GetMeResponseSchema, ListCategoriesResponseSchema } from '@studyverse/proto'

const origin = 'http://127.0.0.1:6872'
const privateValue = 'private-input-never-exported'
const errorMessage = `Request failed\n${privateValue}\n    at ${privateValue} (https://example.test/private)`

async function mockReads(context: BrowserContext) {
  const cors = {
    'access-control-allow-origin': origin,
    'access-control-allow-credentials': 'true',
    'access-control-allow-methods': 'GET, POST, OPTIONS',
    'access-control-allow-headers':
      'content-type, connect-protocol-version, connect-timeout-ms, x-user-agent, traceparent, tracestate'
  }
  const session = {
    id: privateValue,
    active: true,
    expires_at: new Date(Date.now() + 3_600_000).toISOString(),
    identity: { id: '11111111-1111-4111-8111-111111111111', traits: { email: `${privateValue}@example.test` } }
  }
  await context.route('**/sessions/whoami', (route) => route.fulfill({ headers: cors, json: session }))
  await context.route('**/api/**', (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/auth/session') return route.fulfill({ json: session })
    if (path.endsWith('/test.ContextService/Ping') && process.env.OTEL_REHEARSAL === '1') return route.continue()
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
            user: { id: 1, username: privateValue, email: `${privateValue}@example.test` },
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
            categories: [{ id: 1, name: 'Telemetry fixture category', color: '#3B82F6' }],
            total: 1
          })
        )
      )
    })
  })
}

test('enabled production browser SDKs export all signals and bounded product events without private input', async ({
  context,
  page
}) => {
  await mockReads(context)
  const signals: Record<string, string[]> = { traces: [], metrics: [], logs: [] }
  await context.route('**/telemetry/v1/**', async (route) => {
    const signal = new URL(route.request().url()).pathname.split('/').at(-1)!
    signals[signal]?.push(route.request().postData() || '')
    if (process.env.OTEL_REHEARSAL === '1') return route.continue()
    return route.fulfill({ json: {} })
  })
  await page.goto(`/?secret=${privateValue}`)
  await expect(page.locator('g#FIRST_FLOOR')).toBeVisible()
  await expect.poll(() => signals.traces.join(''), { timeout: 25_000 }).toContain('map-frontend')
  await page.getByRole('button', { name: 'Switch to floor 2', exact: true }).click()
  await expect(page.locator('g#SECOND_FLOOR')).toBeVisible()
  await page.evaluate(
    (message) => window.dispatchEvent(new ErrorEvent('error', { error: new Error(message) })),
    errorMessage
  )
  const admin = await context.newPage()
  await admin.goto(`/administration/categories?secret=${privateValue}`)
  await expect(admin.getByText('Telemetry fixture category', { exact: true })).toBeVisible()
  await expect.poll(() => signals.traces.join(''), { timeout: 25_000 }).toContain('ems-frontend')
  await admin.evaluate(
    (message) => window.dispatchEvent(new ErrorEvent('error', { error: new Error(message) })),
    errorMessage
  )
  for (const signal of ['traces', 'metrics', 'logs']) {
    await expect.poll(() => signals[signal].join(''), { timeout: 25_000 }).toContain('map-frontend')
    await expect.poll(() => signals[signal].join(''), { timeout: 25_000 }).toContain('ems-frontend')
  }
  expect(signals.metrics.join('')).toContain('studyverse.ui.events')
  expect(signals.metrics.join('')).toContain('floor_change')
  expect(signals.metrics.join('')).toContain('studyverse.web_vital')
  expect(Object.values(signals).flat().join('')).not.toContain(privateValue)

  let traceparent = ''
  page.on('request', (request) => {
    if (request.url().endsWith('/test.ContextService/Ping')) traceparent = request.headers().traceparent || ''
  })
  const status = await page.evaluate(
    async () =>
      (
        await fetch('/api/ems/test.ContextService/Ping', {
          method: 'POST',
          headers: { 'content-type': 'application/json', 'connect-protocol-version': '1' },
          body: '{}'
        })
      ).status
  )
  expect(status).toBe(200)
  expect(traceparent).toMatch(/^00-[a-f\d]{32}-[a-f\d]{16}-01$/)
  if (process.env.OTEL_REHEARSAL === '1') {
    await mkdir('test-results', { recursive: true })
    await writeFile('test-results/signoz-browser-trace-id.txt', traceparent.split('-')[1])
    await expect.poll(() => signals.traces.join(''), { timeout: 10_000 }).toContain(traceparent.split('-')[1])
  }
})

test('collector unavailability leaves production navigation and theme interactions usable', async ({
  context,
  page
}) => {
  await mockReads(context)
  let exports = 0
  await context.route('**/telemetry/v1/**', (route) => {
    exports++
    return route.fulfill({ status: 503, json: { error: 'Collector unavailable' } })
  })
  await page.goto('/')
  await expect(page.locator('g#FIRST_FLOOR')).toBeVisible()
  await expect.poll(() => exports, { timeout: 20_000 }).toBeGreaterThan(0)
  await page.getByRole('button', { name: 'Switch to floor 3', exact: true }).click()
  await expect(page.locator('g#THIRD_FLOOR')).toBeVisible()
  const admin = await context.newPage()
  await admin.goto('/administration/categories')
  await expect(admin.getByText('Telemetry fixture category', { exact: true })).toBeVisible()
  await admin.getByRole('button', { name: 'Theme: Dark' }).click()
  await admin.getByRole('menuitemradio', { name: 'Light', exact: true }).click()
  await expect(admin.locator('html')).toHaveClass(/light/)
})
