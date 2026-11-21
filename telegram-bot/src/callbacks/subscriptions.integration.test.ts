import assert from 'node:assert/strict'
import { test } from 'node:test'
import { eq } from 'drizzle-orm'
import { Bot } from 'grammy'

test('Grammy subscription and reminder preference callbacks persist changes only for the sender', {
  skip: !process.env.QA_DATABASE_URL
}, async () => {
  const databaseUrl = new URL(process.env.QA_DATABASE_URL || '')
  assert.equal(databaseUrl.hostname, '127.0.0.1')
  assert.equal(databaseUrl.pathname, '/qa_bot_tests')
  process.env.DATABASE_URL = databaseUrl.href
  process.env.TELEGRAM_BOT_TOKEN = '123456:synthetic-test-token'
  process.env.INTERNAL_REMINDER_SECRET = 'qa-internal-only-secret-32-characters'
  const { db, schema, sqlClient } = await import('../db')
  const { setupCallbacks } = await import('./index')
  const calls: string[] = []
  const mockTelegram = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch: (request) => {
      calls.push(new URL(request.url).pathname.split('/').pop() || '')
      return Response.json({ ok: true, result: true })
    }
  })
  const bot = new Bot('123456:synthetic-test-token', {
    botInfo: {
      id: 123456,
      is_bot: true,
      first_name: 'QA Bot',
      username: 'qa_only_bot',
      can_join_groups: false,
      can_read_all_group_messages: false,
      supports_inline_queries: false,
      can_connect_to_business: false,
      has_main_web_app: false,
      has_topics_enabled: false,
      allows_users_to_create_topics: false
    },
    client: { apiRoot: `http://127.0.0.1:${mockTelegram.port}` }
  })
  setupCallbacks(bot)
  const telegramId = 801
  const otherId = 802
  let updateId = 1
  const callback = (data: string) =>
    bot.handleUpdate({
      update_id: updateId++,
      callback_query: {
        id: `qa-${updateId}`,
        from: { id: telegramId, is_bot: false, first_name: 'QA' },
        chat_instance: 'qa-only',
        data,
        message: { message_id: 1, date: 1, chat: { id: telegramId, type: 'private', first_name: 'QA' } }
      }
    })
  const readUser = async () =>
    (await db.select().from(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, telegramId)))[0]
  try {
    await db.delete(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, telegramId))
    await db.insert(schema.telegramUsers).values({ telegramId: otherId, groupName: 'QA-OTHER' })
    await callback('group:QA-ONLY')
    assert.equal((await readUser()).groupName, 'QA-ONLY')
    assert.equal((await readUser()).notificationsEnabled, true)
    await callback('settings:toggle_notifications')
    assert.equal((await readUser()).notificationsEnabled, false)
    await callback('group:QA-UPDATED')
    assert.equal((await readUser()).groupName, 'QA-UPDATED')
    assert.equal((await readUser()).notificationsEnabled, false, 'group changes preserve an opted-out subscriber')
    await callback('settings:toggle_notifications')
    assert.equal((await readUser()).notificationsEnabled, true)
    await callback('reminders:toggle:15_min')
    assert.equal((await readUser()).reminder15Min, true)
    await callback('reminders:toggle:hour_before')
    assert.equal((await readUser()).reminderHourBefore, false)
    const beforeUnknown = await readUser()
    await callback('reminders:toggle:unknown')
    assert.deepEqual(await readUser(), beforeUnknown)
    const [other] = await db.select().from(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, otherId))
    assert.equal(other.groupName, 'QA-OTHER')
    assert.equal(other.reminder15Min, false)
    const originalFetch = globalThis.fetch
    const cancellationRequests: Array<{ secret: string | null; sender: string | null; body: unknown }> = []
    globalThis.fetch = Object.assign(
      async (input: unknown, init?: RequestInit) => {
        const headers = new Headers(init?.headers)
        cancellationRequests.push({
          secret: headers.get('X-Internal-Secret'),
          sender: headers.get('X-Telegram-User-Id'),
          body: JSON.parse(await new Request(String(input), init).text())
        })
        return Response.json({ success: true })
      },
      { preconnect: () => {} }
    )
    try {
      await callback('reminder:cancel:37')
      assert.deepEqual(cancellationRequests, [
        {
          secret: process.env.INTERNAL_REMINDER_SECRET,
          sender: String(telegramId),
          body: { eventId: 37 }
        }
      ])
    } finally {
      globalThis.fetch = originalFetch
    }
    assert.ok(calls.includes('answerCallbackQuery') && calls.includes('editMessageText'))
  } finally {
    for (const id of [telegramId, otherId])
      await db.delete(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, id))
    mockTelegram.stop()
    await sqlClient.end()
  }
})
