import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'

test('lesson reminders reach Telegram once for the enabled group subscriber', {
  skip: !process.env.QA_DATABASE_URL
}, async () => {
  const databaseUrl = new URL(process.env.QA_DATABASE_URL || '')
  assert.equal(databaseUrl.hostname, '127.0.0.1')
  assert.ok(['/prelaunch_bot', '/qa_bot_tests'].includes(databaseUrl.pathname))
  process.env.DATABASE_URL = databaseUrl.href
  process.env.TELEGRAM_BOT_TOKEN = '123456:synthetic-test-token'
  const startsAt = new Date(Date.now() + 14.5 * 60_000)
  const schedule = Bun.serve({
    hostname: '127.0.0.1',
    port: 0,
    fetch: () =>
      Response.json({
        groupName: 'QA-ONLY',
        events: [
          {
            title: 'Prelaunch lesson',
            teacher: 'QA Teacher',
            start: startsAt.toISOString(),
            end: new Date(startsAt.getTime() + 60 * 60_000).toISOString(),
            lessonType: 'Practice',
            location: 'QA room'
          }
        ]
      })
  })
  process.env.MAP_APP_URL = `http://127.0.0.1:${schedule.port}`
  const { db, schema, sqlClient } = await import('../db')
  const { checkAndNotify } = await import('./scheduler')
  const sent: Array<{ chatId: number | string; text: string }> = []
  const bot = new Bot('123456:synthetic-test-token')
  bot.api.sendMessage = async (chatId, text) => {
    sent.push({ chatId, text })
    return { message_id: 42, date: 1, chat: { id: 101, type: 'private', first_name: 'QA' }, text }
  }
  let failDelivery = false
  let attempts = 0
  const sendMessage = bot.api.sendMessage
  bot.api.sendMessage = async (...args) => {
    attempts++
    if (failDelivery) throw new Error('Mock Telegram temporary outage')
    return sendMessage(...args)
  }
  try {
    await db.delete(schema.notificationDeliveries)
    await db.delete(schema.telegramUsers)
    await db.insert(schema.telegramUsers).values([
      { telegramId: 101, groupName: 'QA-ONLY', notificationsEnabled: true },
      { telegramId: 102, groupName: 'QA-ONLY', notificationsEnabled: false },
      { telegramId: 103, notificationsEnabled: true }
    ])
    const dependencies = {
      fetchSchedule: async () => ({
        group: 'QA-ONLY',
        days: [{
          day: 'QA',
          items: [{
            subject: 'Prelaunch lesson', tutor: 'QA Teacher', startsAt: startsAt.toISOString(),
            classtimeDay: 'QA', classtimeTime: '10:00–11:00', lessonType: 'Practice', room: 'QA room',
            group: 'QA-ONLY', teamsMeetingUrl: ''
          }]
        }]
      })
    }
    await checkAndNotify(bot, dependencies)
    await checkAndNotify(bot, dependencies)
    assert.equal(sent.length, 1)
    assert.equal(sent[0].chatId, 101)
    assert.match(sent[0].text, /Prelaunch lesson/)
    const deliveries = await db.select().from(schema.notificationDeliveries)
    assert.equal(deliveries.length, 1)
    assert.equal(deliveries[0].status, 'sent')
    await db.delete(schema.notificationDeliveries)
    failDelivery = true
    await checkAndNotify(bot)
    assert.equal((await db.select().from(schema.notificationDeliveries))[0].status, 'failed')
    failDelivery = false
    await Promise.all([checkAndNotify(bot), checkAndNotify(bot)])
    assert.equal(sent.length, 2, 'a failed reminder should retry successfully once')
    assert.equal(attempts, 3, 'concurrent retry checks should claim only one delivery')
    const [retried] = await db.select().from(schema.notificationDeliveries)
    assert.equal(retried.status, 'sent')
    assert.equal(retried.errorMessage, null)
  } finally {
    schedule.stop()
    await sqlClient.end()
  }
})
