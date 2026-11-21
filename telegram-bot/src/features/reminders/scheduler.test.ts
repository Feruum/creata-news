import assert from 'node:assert/strict'
import { test } from 'node:test'
import type { PendingReminder } from '@studyverse/proto'
import { Bot } from 'grammy'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { processReminderType } = await import('./scheduler')

test('failed Telegram deliveries stay pending while successful deliveries are acknowledged', async () => {
  const bot = new Bot('123456:synthetic-test-token')
  bot.api.sendMessage = async (chatId) => {
    if (chatId === 101) throw new Error('Temporary Telegram outage')
    return { message_id: 42, date: 1, chat: { id: 102, type: 'private', first_name: 'Test' }, text: 'test' }
  }
  const acknowledged: Array<number | undefined> = []
  await processReminderType(bot, 'hour_before', {
    client: {
      getPendingReminders: async () => ({
        $typeName: 'events.v1.GetPendingRemindersResponse',
        reminders: [101, 102].map(
          (id): PendingReminder => ({
            $typeName: 'events.v1.PendingReminder',
            eventId: id,
            userId: id,
            telegramId: BigInt(id),
            reminderType: 'hour_before',
            eventTitle: 'Campus Club test',
            startTime: '2026-10-05T07:00:00Z',
            endTime: '2026-10-05T08:00:00Z',
            userFirstName: 'Test',
            organizationTitle: 'Campus Club',
            slug: 'test',
            eventDescription: ''
          })
        )
      }),
      markRemindersSent: async (request) => {
        acknowledged.push(...(request.reminders || []).map((reminder) => reminder.eventId))
        return { $typeName: 'events.v1.MarkRemindersSentResponse', marked: 1 }
      }
    },
    loadPreferences: async () =>
      [101, 102].map((telegramId) => ({
        telegramId,
        reminderDayBefore: true,
        reminderHourBefore: true,
        reminder15Min: true
      }))
  })
  assert.deepEqual(acknowledged, [102])
})
