import assert from 'node:assert/strict'
import { test } from 'node:test'
import { eq } from 'drizzle-orm'
import { Bot } from 'grammy'

test('/start initializes event preferences before group selection and preserves opt-outs', {
  skip: !process.env.QA_DATABASE_URL
}, async () => {
  const databaseUrl = new URL(process.env.QA_DATABASE_URL || '')
  assert.equal(databaseUrl.hostname, '127.0.0.1')
  assert.equal(databaseUrl.pathname, '/qa_bot_tests')
  process.env.DATABASE_URL = databaseUrl.href
  process.env.TELEGRAM_BOT_TOKEN = '123456:synthetic-test-token'
  const replies: string[] = []
  const menus: Array<{ chat_id: number; menu_button: { type: string } }> = []
  const groupsServer = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch: async (request) => {
      if (request.method === 'GET') return Response.json(['QA-GROUP'])
      const method = new URL(request.url).pathname.split('/').pop()
      if (method === 'setChatMenuButton') {
        menus.push((await request.json()) as { chat_id: number; menu_button: { type: string } })
        return Response.json({ ok: true, result: true })
      }
      assert.equal(method, 'sendMessage')
      const body = (await request.json()) as { chat_id: number; text: string }
      replies.push(body.text)
      return Response.json({
        ok: true,
        result: {
          message_id: 1,
          date: 1,
          chat: { id: body.chat_id, type: 'private', first_name: 'QA' },
          text: body.text
        }
      })
    }
  })
  process.env.MAP_APP_URL = `http://127.0.0.1:${groupsServer.port}`
  const { db, schema, sqlClient } = await import('../db')
  const { setupCommands } = await import('./index')
  const bot = new Bot('123456:synthetic-test-token', {
    client: { apiRoot: 'http://127.0.0.1:' + groupsServer.port },
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
    }
  })
  setupCommands(bot)
  const telegramId = 901
  let updateId = 1
  const command = (text: string) =>
    bot.handleUpdate({
      update_id: updateId++,
      message: {
        message_id: updateId,
        date: 1,
        chat: { id: telegramId, type: 'private', first_name: 'QA' },
        from: { id: telegramId, is_bot: false, first_name: 'QA' },
        text,
        entities: [{ type: 'bot_command', offset: 0, length: text.length }]
      }
    })
  const readUser = async () =>
    (await db.select().from(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, telegramId)))[0]
  try {
    await db.delete(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, telegramId))
    await command('/start')
    assert.deepEqual(menus, [{ chat_id: telegramId, menu_button: { type: 'commands' } }])
    const created = await readUser()
    assert.ok(created, '/start must create preferences without requiring a schedule group')
    assert.equal(created.groupName, null)
    assert.equal(created.reminder15Min, false, 'new users retain the explicit opt-in default')
    await command('/reminders')
    assert.ok(replies.some((text) => text.includes('Event Reminder Preferences')))
    await db
      .update(schema.telegramUsers)
      .set({
        groupName: 'QA-SAVED',
        reminder15Min: true,
        reminderHourBefore: false,
        notificationsEnabled: false
      })
      .where(eq(schema.telegramUsers.telegramId, telegramId))
    await command('/start')
    assert.equal(menus.length, 2, 'returning users also refresh their private chat menu')
    const existing = await readUser()
    assert.equal(existing.groupName, 'QA-SAVED')
    assert.equal(existing.reminder15Min, true)
    assert.equal(existing.reminderHourBefore, false)
    assert.equal(existing.notificationsEnabled, false)
  } finally {
    await db.delete(schema.telegramUsers).where(eq(schema.telegramUsers.telegramId, telegramId))
    groupsServer.stop()
    await sqlClient.end()
  }
})
