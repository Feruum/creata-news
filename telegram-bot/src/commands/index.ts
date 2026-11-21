import { eq } from 'drizzle-orm'
import type { Bot, Context } from 'grammy'
import { InlineKeyboard } from 'grammy'
import { db, schema } from '../db'
import { setupEventsCommand } from '../features/events'
import { setupRemindersCommand } from '../features/reminders'
import { env } from '../lib/env'
import { createMiniAppButton, getMiniAppMenuButton } from '../lib/mini-app'
import { generateMapLink } from '../services/map'
import { fetchGroupSchedule, fetchGroups, getTodayLessons } from '../services/schedule'

const GROUPS_PER_PAGE = 8

export function setupCommands(bot: Bot<Context>) {
  bot.command('start', handleStart)
  bot.command('map', handleMap)
  bot.command('settings', handleSettings)
  bot.command('schedule', handleSchedule)
  bot.command('help', handleHelp)
  bot.command('test', handleTestNotification)
  setupEventsCommand(bot)
  setupRemindersCommand(bot)
}

async function handleStart(ctx: Context) {
  const telegramId = ctx.from?.id
  if (!telegramId) return
  await refreshMiniAppMenu(ctx)

  await db
    .insert(schema.telegramUsers)
    .values({
      telegramId,
      username: ctx.from?.username,
      firstName: ctx.from?.first_name,
      lastName: ctx.from?.last_name
    })
    .onConflictDoNothing({ target: schema.telegramUsers.telegramId })

  const existingUser = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })

  if (existingUser?.groupName) {
    await ctx.reply(
      `Welcome back! Your group is set to *${existingUser.groupName}*.\n\n` +
        `Use /schedule to view today's lessons or /settings to change your group.`,
      {
        parse_mode: 'Markdown',
        reply_markup: new InlineKeyboard().add(createMiniAppButton(env.MAP_APP_URL, ctx.chat?.type))
      }
    )
    return
  }

  await ctx.reply(
    'Welcome to the Schedule Bot! 📚\n\n' +
      'I will notify you 15 minutes before each lesson with:\n' +
      '• Subject and tutor info\n' +
      '• Room location with map link\n\n' +
      'Open the campus map below or select your group for lesson notifications:',
    { reply_markup: new InlineKeyboard().add(createMiniAppButton(env.MAP_APP_URL, ctx.chat?.type)) }
  )

  await sendGroupSelection(ctx, 0)
}

async function handleMap(ctx: Context) {
  await refreshMiniAppMenu(ctx)
  await ctx.reply('Карта кампуса, расписание и события:', {
    reply_markup: new InlineKeyboard().add(createMiniAppButton(env.MAP_APP_URL, ctx.chat?.type))
  })
}

async function refreshMiniAppMenu(ctx: Context) {
  if (ctx.chat?.type !== 'private') return
  try {
    await ctx.api.setChatMenuButton({
      chat_id: ctx.chat.id,
      menu_button: getMiniAppMenuButton(env.MAP_APP_URL)
    })
  } catch {
    // Keep the inline launch usable even if Telegram cannot update the menu.
    console.warn('Failed to refresh the private chat Mini App menu')
  }
}

async function handleSettings(ctx: Context) {
  const keyboard = new InlineKeyboard()
    .text('Change Group', 'settings:change_group')
    .row()
    .text('Toggle Notifications', 'settings:toggle_notifications')

  await ctx.reply('Settings:', { reply_markup: keyboard })
}

async function handleSchedule(ctx: Context) {
  const telegramId = ctx.from?.id
  if (!telegramId) return
  const keyboard = new InlineKeyboard().add(createMiniAppButton(env.MAP_APP_URL, ctx.chat?.type))

  const user = await db.query.telegramUsers.findFirst({
    where: eq(schema.telegramUsers.telegramId, telegramId)
  })

  if (!user?.groupName) {
    await ctx.reply('Please set your group first using /start', { reply_markup: keyboard })
    return
  }

  try {
    const schedule = await fetchGroupSchedule(user.groupName)
    const todayLessons = getTodayLessons(schedule)

    if (todayLessons.length === 0) {
      await ctx.reply('No lessons scheduled for today! 🎉', { reply_markup: keyboard })
      return
    }

    let message = `📅 *Today's Schedule for ${user.groupName}*\n\n`

    for (const lesson of todayLessons) {
      const mapLink = generateMapLink(lesson.mapRoomId || '')
      message += `⏰ *${lesson.classtimeTime}*\n`
      message += `📖 ${lesson.subject}\n`
      message += `👨‍🏫 ${lesson.tutor}\n`
      message += `🏛 [${lesson.room}](${mapLink}) (${lesson.lessonType})\n\n`
    }

    await ctx.reply(message, {
      parse_mode: 'Markdown',
      link_preview_options: { is_disabled: true },
      reply_markup: keyboard
    })
  } catch (error) {
    console.error('Failed to fetch schedule:', error)
    await ctx.reply('Failed to fetch schedule. Please try again later.', { reply_markup: keyboard })
  }
}

async function handleHelp(ctx: Context) {
  await ctx.reply(
    '*Schedule Bot Help* 📚\n\n' +
      '*Commands:*\n' +
      '/start - Start the bot and select your group\n' +
      '/map - Open the campus map, schedule and events\n' +
      "/schedule - View today's schedule\n" +
      '/events - View upcoming campus events\n' +
      '/reminders - Manage event reminder preferences\n' +
      '/settings - Change group or notification settings\n' +
      '/test - Send a test notification\n' +
      '/help - Show this help message\n\n' +
      '*Features:*\n' +
      '• Automatic notifications 15 minutes before each lesson\n' +
      '• Event reminders (24h, 1h, 15min before)\n' +
      '• Direct map links to find your classroom\n' +
      '• View full daily schedule anytime\n' +
      '• Browse upcoming campus events',
    { parse_mode: 'Markdown' }
  )
}

async function handleTestNotification(ctx: Context) {
  const mapLink = generateMapLink('C1.2.301')

  await ctx.reply(
    `📚 *Upcoming Lesson in 15 minutes!*\n\n` +
      `📖 *Subject:* Test Subject\n` +
      `👨‍🏫 *Tutor:* Test Tutor\n` +
      `🏛 *Room:* [C1.2.301](${mapLink})\n` +
      `📝 *Type:* Lecture\n` +
      `⏰ *Time:* 09:00-09:50\n\n` +
      `🗺 [Open Map](${mapLink})`,
    {
      parse_mode: 'Markdown',
      link_preview_options: { is_disabled: true },
      reply_markup: new InlineKeyboard().add(
        createMiniAppButton(env.MAP_APP_URL, ctx.chat?.type, { roomId: 'C1.2.301' })
      )
    }
  )
}

export async function sendGroupSelection(ctx: Context, page: number) {
  try {
    const allGroups = await fetchGroups()
    const totalPages = Math.ceil(allGroups.length / GROUPS_PER_PAGE)
    const startIdx = page * GROUPS_PER_PAGE
    const pageGroups = allGroups.slice(startIdx, startIdx + GROUPS_PER_PAGE)

    const keyboard = new InlineKeyboard()

    for (let i = 0; i < pageGroups.length; i += 2) {
      if (pageGroups[i + 1]) {
        keyboard.text(pageGroups[i], `group:${pageGroups[i]}`).text(pageGroups[i + 1], `group:${pageGroups[i + 1]}`)
      } else {
        keyboard.text(pageGroups[i], `group:${pageGroups[i]}`)
      }
      keyboard.row()
    }

    if (totalPages > 1) {
      if (page > 0) {
        keyboard.text('⬅️ Previous', `page:${page - 1}`)
      }
      keyboard.text(`${page + 1}/${totalPages}`, 'noop')
      if (page < totalPages - 1) {
        keyboard.text('Next ➡️', `page:${page + 1}`)
      }
    }

    if (ctx.callbackQuery) {
      await ctx.editMessageText('Select your group:', { reply_markup: keyboard })
    } else {
      await ctx.reply('Select your group:', { reply_markup: keyboard })
    }
  } catch (error) {
    console.error('Failed to fetch groups:', error)
    await ctx.reply('Failed to load groups. Please try again later with /start')
  }
}
