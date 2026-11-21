import { Cron } from 'croner'
import type { Bot, Context } from 'grammy'
import { db } from '../../db'
import type { TelegramUser } from '../../db/schema'
import { env } from '../../lib/env'
import { eventsClient } from '../../lib/rpc'
import { withSingletonLease } from '../../services/singleton'
import { formatReminderMessage, getReminderKeyboard } from './templates'

type PreferenceField = 'reminderDayBefore' | 'reminderHourBefore' | 'reminder15Min'

function getPreferenceField(reminderType: string): PreferenceField {
  switch (reminderType) {
    case 'day_before':
      return 'reminderDayBefore'
    case 'hour_before':
      return 'reminderHourBefore'
    case '15_min':
      return 'reminder15Min'
    default:
      return 'reminderDayBefore'
  }
}

interface SentReminderEntry {
  eventId: number
  userId: number
  reminderType: string
  telegramMessageId?: bigint
}

interface ReminderDependencies {
  client: Pick<typeof eventsClient, 'getPendingReminders' | 'markRemindersSent'>
  loadPreferences: () => Promise<Array<Pick<TelegramUser, 'telegramId' | PreferenceField>>>
}

export async function processReminderType(
  bot: Bot<Context>,
  reminderType: string,
  dependencies: ReminderDependencies = {
    client: eventsClient,
    loadPreferences: () => db.query.telegramUsers.findMany()
  }
): Promise<void> {
  const response = await dependencies.client.getPendingReminders({ reminderType })
  const reminders = response.reminders
  if (reminders.length === 0) return

  console.log(`Processing ${reminders.length} ${reminderType} reminders`)

  const users = await dependencies.loadPreferences()
  const userPrefs = new Map(users.map((user) => [user.telegramId, user]))

  const prefField = getPreferenceField(reminderType)
  const sentReminders: SentReminderEntry[] = []

  for (const reminder of reminders) {
    const telegramId = Number(reminder.telegramId)
    if (!Number.isFinite(telegramId)) {
      sentReminders.push({
        eventId: reminder.eventId,
        userId: reminder.userId,
        reminderType: reminder.reminderType
      })
      continue
    }

    const userPref = userPrefs.get(telegramId)
    if (!userPref) continue

    if (!userPref[prefField]) {
      sentReminders.push({
        eventId: reminder.eventId,
        userId: reminder.userId,
        reminderType: reminder.reminderType
      })
      continue
    }

    try {
      const message = formatReminderMessage(reminderType, reminder)
      const keyboard = getReminderKeyboard(reminder.eventId, reminder.slug)
      const result = await bot.api.sendMessage(telegramId, message, {
        parse_mode: 'HTML',
        reply_markup: keyboard,
        link_preview_options: { is_disabled: true }
      })

      sentReminders.push({
        eventId: reminder.eventId,
        userId: reminder.userId,
        reminderType: reminder.reminderType,
        telegramMessageId: BigInt(result.message_id)
      })
    } catch (error) {
      console.error(`Failed to send reminder to ${telegramId}:`, error)
    }
  }

  if (sentReminders.length > 0) {
    await dependencies.client.markRemindersSent({ reminders: sentReminders })
  }
  const delivered = sentReminders.filter((entry) => entry.telegramMessageId !== undefined).length
  console.log(`Delivered ${delivered} ${reminderType} reminders; skipped ${sentReminders.length - delivered}`)
}

async function runReminderCheck(bot: Bot<Context>): Promise<void> {
  try {
    await processReminderType(bot, 'day_before')
    await processReminderType(bot, 'hour_before')
    await processReminderType(bot, '15_min')
  } catch (error) {
    console.error('Event reminder scheduler error:', error)
  }
}

export function startEventReminderScheduler(bot: Bot<Context>) {
  new Cron('*/5 * * * *', { timezone: env.TIMEZONE, protect: true }, async () => {
    await withSingletonLease(7_302_002, () => runReminderCheck(bot))
  })
  console.log('Event reminder scheduler started (every 5 minutes)')
}
