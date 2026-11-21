import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createClient } from '@connectrpc/connect'
import { EventsService } from '@studyverse/proto'
import { createBackendTransport } from './rpc-transport'

test('only internal reminder RPCs carry the backend secret', async () => {
  const requests: Array<{ path: string; secret: string | null }> = []
  const client = createClient(EventsService, createBackendTransport({
    baseUrl: 'https://backend.example',
    internalReminderSecret: 'test-internal-secret',
    fetch: Object.assign(async (input: string | URL | Request, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
      requests.push({ path: new URL(url).pathname, secret: new Headers(init?.headers).get('X-Internal-Secret') })
      return new Response('{}', { headers: { 'Content-Type': 'application/json' } })
    }, { preconnect: () => {} })
  }))
  await client.getPendingReminders({ reminderType: 'hour_before' })
  await client.markRemindersSent({ reminders: [] })
  await client.listEvents({})
  assert.deepEqual(requests.map((request) => request.secret), ['test-internal-secret', 'test-internal-secret', null])
})
