import { spyOn } from 'bun:test'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { setupReminderCallbacks } = await import('./callbacks')
const { registrationsClient } = await import('../../lib/rpc')
const { env } = await import('../../lib/env')

test('legacy cancel callbacks open an event with authentication instructions and never cancel as an arbitrary user', async () => {
  const originalSecret = env.INTERNAL_REMINDER_SECRET
  env.INTERNAL_REMINDER_SECRET = ''
  const cancelled = spyOn(registrationsClient, 'cancelRegistration').mockResolvedValue({
    $typeName: 'events.v1.CancelRegistrationResponse',
    success: true
  })
  const messages: Array<{ method: string; payload: unknown }> = []
  const bot = new Bot('123456:synthetic-test-token', {
    botInfo: {
      id: 123456,
      is_bot: true,
      first_name: 'Test',
      username: 'test_bot',
      can_join_groups: true,
      can_read_all_group_messages: false,
      supports_inline_queries: false,
      can_connect_to_business: false,
      has_main_web_app: false,
      has_topics_enabled: false,
      allows_users_to_create_topics: false
    }
  })
  bot.api.config.use(async (previous, method, payload) => {
    messages.push({ method, payload })
    return { ok: true, result: true } as Awaited<ReturnType<typeof previous<typeof method>>>
  })
  setupReminderCallbacks(bot)
  try {
    await bot.handleUpdate({
      update_id: 1,
      callback_query: {
        id: 'test',
        from: { id: 101, is_bot: false, first_name: 'Test' },
        chat_instance: 'test',
        data: 'reminder:cancel:42',
        message: {
          message_id: 1,
          date: 1,
          chat: { id: 101, type: 'private', first_name: 'Test' },
          text: 'Event reminder'
        }
      }
    })
    assert.equal(cancelled.mock.calls.length, 0)
    const reply = JSON.stringify(messages)
    assert.match(reply, /sign in/i)
    assert.match(reply, /web_app|url/)
    assert.doesNotMatch(reply, /has been cancelled|Registration cancelled/)
  } finally {
    env.INTERNAL_REMINDER_SECRET = originalSecret
    cancelled.mockRestore()
  }
})
