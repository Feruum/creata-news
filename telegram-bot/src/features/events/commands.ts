import type { Bot, Context } from 'grammy'
import { InlineKeyboard } from 'grammy'
import { env } from '../../lib/env'
import { createMiniAppButton } from '../../lib/mini-app'
import { fetchUpcomingEvents, formatEventMessage } from '../../services/events'

const EVENTS_PER_PAGE = 3

export function setupEventsCommand(bot: Bot<Context>) {
  bot.command('events', handleEvents)
}

async function handleEvents(ctx: Context) {
  try {
    await showEventsPage(ctx, 0)
  } catch (error) {
    console.error('Failed to handle /events command:', error)
    await ctx.reply('Failed to fetch events. Please try again later.')
  }
}

export async function showEventsPage(ctx: Context, page: number) {
  try {
    const offset = page * EVENTS_PER_PAGE
    const response = await fetchUpcomingEvents(EVENTS_PER_PAGE, offset)
    const { events, total } = response

    if (events.length === 0) {
      await ctx.reply('No upcoming events found. 📭')
      return
    }

    let message = '📅 *Upcoming Campus Events*\n\n'

    for (const event of events) {
      message += `${formatEventMessage(event)}\n\n`
    }

    const totalPages = Math.ceil(total / EVENTS_PER_PAGE)
    const keyboard = new InlineKeyboard()

    for (const event of events) {
      keyboard
        .add(
          createMiniAppButton(
            env.MAP_APP_URL,
            ctx.chat?.type,
            { eventSlug: event.slug || event.id },
            `📌 ${event.title}`
          )
        )
        .row()
    }

    if (page > 0) {
      keyboard.text('⬅️ Previous', `events:page:${page - 1}`)
    }

    keyboard.text(`${page + 1}/${totalPages}`, 'noop')

    if (page < totalPages - 1) {
      keyboard.text('Next ➡️', `events:page:${page + 1}`)
    }

    if (ctx.callbackQuery) {
      await ctx.editMessageText(message, {
        parse_mode: 'Markdown',
        reply_markup: keyboard,
        link_preview_options: { is_disabled: true }
      })
    } else {
      await ctx.reply(message, {
        parse_mode: 'Markdown',
        reply_markup: keyboard,
        link_preview_options: { is_disabled: true }
      })
    }
  } catch (error) {
    console.error('Failed to fetch events page:', error)
    await ctx.reply('Failed to load events. Please try again later.')
  }
}
