import assert from 'node:assert/strict'
import { test } from 'node:test'

process.env.TELEGRAM_BOT_TOKEN ||= '123456:synthetic-test-token'
process.env.INTERNAL_REMINDER_SECRET ||= 'audit-internal-notification-secret-32'
const { bot } = await import('../bot')
const { server } = await import('./index')

test('internal notification endpoints reject missing and wrong credentials before sending', async () => {
  const original = bot.api.sendMessage
  let sent = 0
  bot.api.sendMessage = async (chatId, text) => {
    sent += 1
    return { message_id: 1, date: 1, chat: { id: Number(chatId), type: 'private', first_name: 'Test' }, text }
  }
  try {
    for (const path of ['/notify', '/notify/bulk', '/notify/', '/notify/bulk/']) {
      for (const secret of [undefined, 'wrong-secret']) {
        const headers: Record<string, string> = { 'Content-Type': 'application/json' }
        if (secret) headers['X-Internal-Secret'] = secret
        const body = !path.startsWith('/notify/bulk')
          ? { telegramId: 101, message: 'Test' }
          : { notifications: [{ telegramId: 101, message: 'Test' }] }
        const response = await server.handle(
          new Request(`http://localhost${path}`, { method: 'POST', headers, body: JSON.stringify(body) })
        )
        assert.equal(response.status, 401)
      }
    }
    assert.equal(sent, 0)
  } finally {
    bot.api.sendMessage = original
  }
})

test('public Telegram webhook retains its own credential and does not require the internal notification secret', async () => {
  const { env } = await import('../lib/env')
  const original = bot.handleUpdate
  let handled = 0
  bot.handleUpdate = async () => {
    handled += 1
  }
  try {
    const response = await server.handle(
      new Request('http://localhost/telegram/webhook', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Telegram-Bot-Api-Secret-Token': env.TELEGRAM_WEBHOOK_SECRET },
        body: JSON.stringify({ update_id: 1 })
      })
    )
    assert.equal(response.status, 200)
    assert.equal(handled, 1)
  } finally {
    bot.handleUpdate = original
  }
})

test('public health routes remain available without an internal notification credential', async () => {
  assert.equal((await server.handle(new Request('http://localhost/health/live'))).status, 200)
})
