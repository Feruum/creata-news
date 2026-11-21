import { eq } from 'drizzle-orm'
import type { Bot, Context } from 'grammy'
import { InlineKeyboard } from 'grammy'
import { db, schema } from '../../db'
import { env } from '../../lib/env'
import { createMiniAppButton } from '../../lib/mini-app'
import { registrationsClient } from '../../lib/rpc'

export function setupReminderCallbacks(bot: Bot<Context>) {
  bot.callbackQuery(/^reminder:confirm:(\d+)$/, handleReminderConfirm)
  bot.callbackQuery(/^reminder:cancel:(\d+)$/, handleReminderCancel)
  bot.callbackQuery(/^reminders:toggle:(.+)$/, handleReminderToggle)
}

async function handleReminderConfirm(ctx: Context) {
  await ctx.answerCallbackQuery({ text: '✅ See you there!' })
}

async function handleReminderCancel(ctx: Context) {
  const match = ctx.callbackQuery?.data?.match(/^reminder:cancel:(\d+)$/)
  if (!match) return

  const eventId = Number.parseInt(match[1], 10)
  const telegramId = ctx.from?.id
  if (!telegramId) return
  if (!env.INTERNAL_REMINDER_SECRET) {
    await ctx.answerCallbackQuery({ text: 'Open the event and sign in to cancel your registration.' })
    await ctx.reply('Open the event in the app, sign in, and choose Cancel registration.', {
      reply_markup: new InlineKeyboard().add(createMiniAppButton(
        env.MAP_APP_URL, ctx.chat?.type, { eventSlug: String(eventId) }, 'Open event to cancel registration'
      ))
    })
    return
  }
  try {
    await registrationsClient.cancelRegistration({ eventId }, {
      headers: {
        'X-Internal-Secret': env.INTERNAL_REMINDER_SECRET,
        'X-Telegram-User-Id': String(telegramId)
      },
      timeoutMs: 10_000
    })
    await ctx.answerCallbackQuery({ text: '❌ Registration cancelled' })
    await ctx.editMessageText('Your registration has been cancelled.', { parse_mode: 'HTML' })
  } catch (error) {
    console.error(`Failed to cancel registration for event ${eventId}:`, error)
    await ctx.answerCallbackQuery({ text: 'Failed to cancel. Try again later.' })
  }
}

async function handleReminderToggle(ctx: Context) {
  const match = ctx.callbackQuery?.data?.match(/^reminders:toggle:(.+)$/)
  if (!match) return

  const field = match[1]
  const telegramId = ctx.from?.id
  if (!telegramId) return

  const user = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })

  if (!user) {
    await ctx.answerCallbackQuery({ text: 'Please set up your account first with /start' })
    return
  }

  let newValue: boolean
  let label: string

  switch (field) {
    case 'day_before':
      newValue = !user.reminderDayBefore
      label = '24h before'
      await db
        .update(schema.telegramUsers)
        .set({ reminderDayBefore: newValue, updatedAt: new Date().toISOString() })
        .where(eq(schema.telegramUsers.telegramId, telegramId))
      break
    case 'hour_before':
      newValue = !user.reminderHourBefore
      label = '1h before'
      await db
        .update(schema.telegramUsers)
        .set({ reminderHourBefore: newValue, updatedAt: new Date().toISOString() })
        .where(eq(schema.telegramUsers.telegramId, telegramId))
      break
    case '15_min':
      newValue = !user.reminder15Min
      label = '15min before'
      await db
        .update(schema.telegramUsers)
        .set({ reminder15Min: newValue, updatedAt: new Date().toISOString() })
        .where(eq(schema.telegramUsers.telegramId, telegramId))
      break
    default:
      await ctx.answerCallbackQuery({ text: 'Unknown preference' })
      return
  }

  const status = newValue ? 'enabled ✅' : 'disabled ❌'
  await ctx.answerCallbackQuery({ text: `${label} reminder ${status}` })

  const updatedUser = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })
  if (!updatedUser) return

  const keyboard = buildPreferencesKeyboard(updatedUser)
  await ctx.editMessageText(formatPreferencesMessage(updatedUser), {
    parse_mode: 'HTML',
    reply_markup: keyboard
  })
}

interface UserPrefs {
  reminderDayBefore: boolean
  reminderHourBefore: boolean
  reminder15Min: boolean
}

export function buildPreferencesKeyboard(user: UserPrefs): InlineKeyboard {
  const dayIcon = user.reminderDayBefore ? '✅' : '❌'
  const hourIcon = user.reminderHourBefore ? '✅' : '❌'
  const minIcon = user.reminder15Min ? '✅' : '❌'

  return new InlineKeyboard()
    .text(`${dayIcon} 24h before`, 'reminders:toggle:day_before')
    .row()
    .text(`${hourIcon} 1h before`, 'reminders:toggle:hour_before')
    .row()
    .text(`${minIcon} 15min before`, 'reminders:toggle:15_min')
}

export function formatPreferencesMessage(user: UserPrefs): string {
  const dayStatus = user.reminderDayBefore ? '✅ On' : '❌ Off'
  const hourStatus = user.reminderHourBefore ? '✅ On' : '❌ Off'
  const minStatus = user.reminder15Min ? '✅ On' : '❌ Off'

  return (
    '<b>🔔 Event Reminder Preferences</b>\n\n' +
    `24 hours before: ${dayStatus}\n` +
    `1 hour before: ${hourStatus}\n` +
    `15 minutes before: ${minStatus}\n\n` +
    'Tap a button to toggle:'
  )
}
