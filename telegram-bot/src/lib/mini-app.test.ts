import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createMiniAppButton, generateMiniAppUrl, getMiniAppMenuButton, mapAppUrlSchema } from './mini-app'

const baseUrl = 'https://map.webspace.cat'

test('private HTTPS launches use Telegram Web App buttons', () => {
  assert.deepEqual(createMiniAppButton(baseUrl, 'private'), {
    text: 'Открыть карту',
    web_app: { url: baseUrl }
  })
})

test('groups and unknown chat types keep ordinary URL buttons', () => {
  for (const chatType of ['group', 'supergroup', 'channel', undefined] as const) {
    assert.deepEqual(createMiniAppButton(baseUrl, chatType), { text: 'Открыть карту', url: baseUrl })
  }
})

test('HTTP development URLs never become Web App buttons', () => {
  assert.deepEqual(createMiniAppButton('http://localhost:3000', 'private'), {
    text: 'Открыть карту',
    url: 'http://localhost:3000'
  })
})

test('contextual room URLs encode room IDs as a single path segment', () => {
  assert.equal(
    generateMiniAppUrl(`${baseUrl}/`, { roomId: 'C1.2.301 / A?#' }),
    `${baseUrl}/room/C1.2.301%20%2F%20A%3F%23`
  )
})

test('contextual event URLs use the singular event route and encode slugs', () => {
  assert.deepEqual(createMiniAppButton(baseUrl, 'private', { eventSlug: 'open day/2026' }, 'Открыть событие'), {
    text: 'Открыть событие',
    web_app: { url: `${baseUrl}/event/open%20day%2F2026` }
  })
})

test('menu opens the HTTPS map and reverts to commands for HTTP development', () => {
  assert.deepEqual(getMiniAppMenuButton(baseUrl), { type: 'web_app', text: 'Открыть карту', web_app: { url: baseUrl } })
  assert.deepEqual(getMiniAppMenuButton('http://localhost:3000'), { type: 'commands' })
})

test('map configuration accepts HTTP(S) and rejects other schemes or malformed URLs', () => {
  for (const value of [baseUrl, 'http://localhost:3000']) assert.equal(mapAppUrlSchema.safeParse(value).success, true)
  for (const value of ['not a url', 'ftp://example.com', 'javascript:alert(1)']) {
    assert.equal(mapAppUrlSchema.safeParse(value).success, false)
  }
})
