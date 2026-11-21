import { eq } from 'drizzle-orm'
import type { Bot, Context } from 'grammy'
import { sendGroupSelection } from '../commands'
import { db, schema } from '../db'
import { showEventsPage } from '../features/events'
import { setupReminderCallbacks } from '../features/reminders'

export function setupCallbacks(bot: Bot<Context>) {
  bot.callbackQuery(/^group:(.+)$/, handleGroupSelect)
  bot.callbackQuery(/^page:(\d+)$/, handlePageNavigation)
  bot.callbackQuery(/^events:page:(\d+)$/, handleEventsPageNavigation)
  bot.callbackQuery('settings:change_group', handleChangeGroup)
  bot.callbackQuery('settings:toggle_notifications', handleToggleNotifications)
  bot.callbackQuery('noop', (ctx) => ctx.answerCallbackQuery())
  setupReminderCallbacks(bot)
}

async function handleGroupSelect(ctx: Context) {
  const telegramId = ctx.from?.id
  if (!telegramId) return

  const match = ctx.callbackQuery?.data?.match(/^group:(.+)$/)
  if (!match) return

  const groupName = match[1]

  await db
    .insert(schema.telegramUsers)
    .values({
      telegramId,
      username: ctx.from?.username,
      firstName: ctx.from?.first_name,
      lastName: ctx.from?.last_name,
      groupName,
      notificationsEnabled: true
    })
    .onConflictDoUpdate({
      target: schema.telegramUsers.telegramId,
      set: {
        groupName,
        username: ctx.from?.username,
        firstName: ctx.from?.first_name,
        lastName: ctx.from?.last_name,
        updatedAt: new Date().toISOString()
      }
    })

  await ctx.answerCallbackQuery({ text: `Group set to ${groupName}!` })
  await ctx.editMessageText(
    `✅ Your group is now set to *${groupName}*!\n\n` +
      `You'll receive notifications 15 minutes before each lesson.\n\n` +
      `Use /schedule to view today's lessons.`,
    { parse_mode: 'Markdown' }
  )
}

async function handlePageNavigation(ctx: Context) {
  const match = ctx.callbackQuery?.data?.match(/^page:(\d+)$/)
  if (!match) return

  const page = Number.parseInt(match[1], 10)
  await ctx.answerCallbackQuery()
  await sendGroupSelection(ctx, page)
}

async function handleChangeGroup(ctx: Context) {
  await ctx.answerCallbackQuery()
  await sendGroupSelection(ctx, 0)
}

async function handleToggleNotifications(ctx: Context) {
  const telegramId = ctx.from?.id
  if (!telegramId) return

  const user = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })

  if (!user) {
    await ctx.answerCallbackQuery({ text: 'Please set your group first with /start' })
    return
  }

  const newValue = !user.notificationsEnabled

  await db
    .update(schema.telegramUsers)
    .set({ notificationsEnabled: newValue, updatedAt: new Date().toISOString() })
    .where(eq(schema.telegramUsers.telegramId, telegramId))

  const status = newValue ? 'enabled ✅' : 'disabled ❌'
  await ctx.answerCallbackQuery({ text: `Notifications ${status}` })
  await ctx.editMessageText(`Notifications are now *${status}*`, { parse_mode: 'Markdown' })
}

async function handleEventsPageNavigation(ctx: Context) {
  const match = ctx.callbackQuery?.data?.match(/^events:page:(\d+)$/)
  if (!match) return

  const page = Number.parseInt(match[1], 10)
  await ctx.answerCallbackQuery()
  await showEventsPage(ctx, page)
}
