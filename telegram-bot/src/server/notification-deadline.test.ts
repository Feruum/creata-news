import assert from 'node:assert/strict'
import { test } from 'node:test'
import { setTimeout as wait } from 'node:timers/promises'
import { autoRetry } from '@grammyjs/auto-retry'
import { Bot } from 'grammy'
import { z } from 'zod'
import { createNotificationRoutes } from './notifications'

const secret = 'local-notification-deadline-secret-32'

function fixture() {
  let attempts = 0
  let delivered = 0
  const telegram = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch: async (request) => {
      assert.ok(new URL(request.url).pathname.endsWith('/sendMessage'))
      const body = z.object({ chat_id: z.number(), text: z.string() }).parse(await request.json())
      attempts++
      if (attempts === 1)
        return Response.json(
          { ok: false, error_code: 429, description: 'Local rate-limit test', parameters: { retry_after: 1 } },
          { status: 429 }
        )
      delivered++
      return Response.json({
        ok: true,
        result: {
          message_id: delivered,
          date: 1,
          chat: { id: body.chat_id, type: 'private', first_name: 'Local QA' },
          text: body.text
        }
      })
    }
  })
  const bot = new Bot('123456:synthetic-deadline-token', { client: { apiRoot: telegram.url.origin } })
  // Use the same retry transformer as the running bot, including retry_after waits.
  bot.api.config.use(autoRetry())
  return { bot, telegram, attempts: () => attempts, delivered: () => delivered }
}

function request(path: string, signal?: AbortSignal) {
  return new Request(`http://localhost${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Internal-Secret': secret },
    body: JSON.stringify(
      path === '/notify'
        ? { telegramId: 101, message: 'Synthetic deadline' }
        : { notifications: [{ telegramId: 101, message: 'Synthetic deadline' }] }
    ),
    signal
  })
}

test('an aborted caller cancels Telegram retries before they can deliver a late notification', async () => {
  const { bot, telegram, attempts, delivered } = fixture()
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 500)
  try {
    const app = createNotificationRoutes(bot.api, secret)
    const response = await app.handle(request('/notify/bulk', controller.signal))
    assert.equal(response.status, 502)
    await wait(1100)
    assert.equal(attempts(), 1)
    assert.equal(delivered(), 0)
  } finally {
    clearTimeout(timer)
    telegram.stop(true)
  }
})

test('single and bulk notification requests stop retries at their delivery deadline', async () => {
  for (const path of ['/notify', '/notify/bulk']) {
    const { bot, telegram, attempts, delivered } = fixture()
    try {
      const app = createNotificationRoutes(bot.api, secret, 500)
      const response = await app.handle(request(path))
      assert.equal(response.status, 502)
      await wait(1100)
      assert.equal(attempts(), 1)
      assert.equal(delivered(), 0)
    } finally {
      telegram.stop(true)
    }
  }
})
