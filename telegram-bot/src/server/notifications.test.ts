import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'
import { createNotificationRoutes, MAX_NOTIFICATION_BODY_BYTES } from './notifications'

const secret = 'audit-internal-notification-secret-32'
function fixture(fail = false, configuredSecret = secret) {
  const bot = new Bot('123456:synthetic-test-token')
  const sent: number[] = []
  bot.api.sendMessage = async (chatId, text) => {
    if (fail) throw new Error('Temporary Telegram failure')
    sent.push(Number(chatId))
    return { message_id: 1, date: 1, chat: { id: Number(chatId), type: 'private', first_name: 'Test' }, text }
  }
  const app = createNotificationRoutes(bot.api, configuredSecret)
  const post = (path: string, body: unknown, key = secret) =>
    app.handle(
      new Request(`http://localhost${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Internal-Secret': key },
        body: JSON.stringify(body)
      })
    )
  return { app, post, sent }
}

test('valid authenticated notification routes send single and bounded bulk requests', async () => {
  const { post, sent } = fixture()
  assert.equal((await post('/notify', { telegramId: 101, message: 'x'.repeat(4096) })).status, 200)
  const notifications = Array.from({ length: 100 }, (_, index) => ({ telegramId: index + 1, message: 'Test' }))
  const response = await post('/notify/bulk', {
    notifications,
    parseMode: 'HTML',
    replyMarkup: { inline_keyboard: [[{ text: 'View', url: 'https://example.test/event' }]] }
  })
  assert.equal(response.status, 200)
  assert.deepEqual(await response.json(), { success: true, sent: 100, failed: 0, total: 100 })
  assert.equal(sent.length, 101)
})

test('bulk counts, messages, Telegram IDs and keyboard fields are bounded before sending', async () => {
  const { post, sent } = fixture()
  const invalid = [
    { notifications: [] },
    { notifications: Array.from({ length: 101 }, () => ({ telegramId: 1, message: 'Test' })) },
    { notifications: [{ telegramId: 1, message: '' }] },
    { notifications: [{ telegramId: 1, message: 'x'.repeat(4097) }] },
    { notifications: [{ telegramId: 1.5, message: 'Test' }] },
    { notifications: [{ telegramId: 0, message: 'Test' }] },
    {
      notifications: [{ telegramId: 1, message: 'Test' }],
      replyMarkup: { inline_keyboard: [[{ text: 'x'.repeat(65), url: 'https://example.test' }]] }
    }
  ]
  for (const body of invalid) assert.equal((await post('/notify/bulk', body)).status, 422)
  assert.equal((await post('/notify', { telegramId: 1, message: 'x'.repeat(4097) })).status, 422)
  assert.equal(sent.length, 0)
})

test('body bounds apply to streamed requests without a content length and malformed JSON does not send', async () => {
  const { app, sent } = fixture()
  const headers = { 'Content-Type': 'application/json', 'X-Internal-Secret': secret }
  const oversized = new Request('http://localhost/notify', {
    method: 'POST',
    headers,
    body: ' '.repeat(MAX_NOTIFICATION_BODY_BYTES + 1)
  })
  assert.equal((await app.handle(oversized)).status, 413)
  assert.equal(
    (await app.handle(new Request('http://localhost/notify', { method: 'POST', headers, body: '{' }))).status,
    400
  )
  assert.equal(sent.length, 0)
})

test('an unconfigured secret never authorizes and Telegram delivery failures return a retryable error', async () => {
  const disabled = fixture(false, '')
  assert.equal((await disabled.post('/notify', { telegramId: 1, message: 'Test' })).status, 401)
  assert.equal(disabled.sent.length, 0)
  const unavailable = fixture(true)
  assert.equal((await unavailable.post('/notify', { telegramId: 1, message: 'Test' })).status, 502)
  const response = await unavailable.post('/notify/bulk', { notifications: [{ telegramId: 1, message: 'Test' }] })
  assert.equal(response.status, 502)
  assert.deepEqual(await response.json(), { success: false, sent: 0, failed: 1, total: 1 })
})
