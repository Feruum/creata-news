import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { processReminderType } = await import('./scheduler')

test('15-minute reminders send only to opted-in users and missing setup remains retryable', async () => {
  const bot = new Bot('123456:synthetic-test-token')
  const sent: Array<number | string> = []
  bot.api.sendMessage = async (chatId, text) => {
    sent.push(chatId)
    return { message_id: 42, date: 1, chat: { id: Number(chatId), type: 'private', first_name: 'QA' }, text }
  }
  const acknowledged: Array<number | undefined> = []
  await processReminderType(bot, '15_min', {
    client: {
      getPendingReminders: async () => ({
        $typeName: 'events.v1.GetPendingRemindersResponse',
        reminders: [101, 102, 103].map((id) => ({
          $typeName: 'events.v1.PendingReminder',
          eventId: id,
          userId: id,
          telegramId: BigInt(id),
          reminderType: '15_min',
          eventTitle: 'QA',
          startTime: '2026-10-02T21:00:00Z',
          endTime: '2026-10-02T22:00:00Z',
          userFirstName: 'QA',
          organizationTitle: 'QA',
          slug: 'qa',
          eventDescription: ''
        }))
      }),
      markRemindersSent: async (request) => {
        acknowledged.push(...(request.reminders || []).map((entry) => entry.eventId))
        return { $typeName: 'events.v1.MarkRemindersSentResponse', marked: 2 }
      }
    },
    loadPreferences: async () => [
      { telegramId: 101, reminderDayBefore: true, reminderHourBefore: true, reminder15Min: true },
      { telegramId: 102, reminderDayBefore: true, reminderHourBefore: true, reminder15Min: false }
    ]
  })
  assert.deepEqual(sent, [101])
  assert.deepEqual(acknowledged, [101, 102], 'missing bot setup must not be recorded as a completed reminder')
})
