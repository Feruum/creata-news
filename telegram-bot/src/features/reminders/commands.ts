import { eq } from 'drizzle-orm'
import type { Bot, Context } from 'grammy'
import { db, schema } from '../../db'
import { buildPreferencesKeyboard, formatPreferencesMessage } from './callbacks'

export function setupRemindersCommand(bot: Bot<Context>) {
  bot.command('reminders', handleReminders)
}

async function handleReminders(ctx: Context) {
  const telegramId = ctx.from?.id
  if (!telegramId) return

  const user = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })

  if (!user) {
    await ctx.reply('Please set up your account first with /start')
    return
  }

  const keyboard = buildPreferencesKeyboard(user)
  await ctx.reply(formatPreferencesMessage(user), {
    parse_mode: 'HTML',
    reply_markup: keyboard
  })
}
