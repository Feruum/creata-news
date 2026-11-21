import { type EventLocation, LocationType } from '@studyverse/proto'
import { InlineKeyboard } from 'grammy'
import type { Chat } from 'grammy/types'
import { env } from '../../lib/env'
import { createMiniAppButton } from '../../lib/mini-app'

interface ReminderData {
  eventTitle: string
  startTime: string
  endTime: string
  organizationTitle: string
  slug: string
  location?: EventLocation
}

function escapeHtml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

function formatLocation(location?: EventLocation): string {
  if (!location) return ''
  if (location.type === LocationType.CAMPUS_ROOM && location.roomName) {
    const floor = location.floor ? `, Floor ${location.floor}` : ''
    return `📍 ${escapeHtml(location.roomName)}${floor}`
  }
  if (location.type === LocationType.EXTERNAL && location.externalAddress) {
    return `📍 ${escapeHtml(location.externalAddress)}`
  }
  if (location.type === LocationType.ONLINE && location.onlineUrl) {
    return `🔗 <a href="${escapeHtml(location.onlineUrl)}">${escapeHtml(location.onlinePlatform || 'Online')}</a>`
  }
  if (location.type === LocationType.HYBRID) {
    const parts: string[] = []
    if (location.roomName) {
      const floor = location.floor ? `, Floor ${location.floor}` : ''
      parts.push(`📍 ${escapeHtml(location.roomName)}${floor}`)
    }
    if (location.onlineUrl) {
      parts.push(`🔗 <a href="${escapeHtml(location.onlineUrl)}">${escapeHtml(location.onlinePlatform || 'Online')}</a>`)
    }
    return parts.join('\n')
  }
  return ''
}

function formatTime(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Almaty'
  })
}

function formatDate(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleDateString('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
    timeZone: 'Asia/Almaty'
  })
}

export function formatReminderMessage(type: string, data: ReminderData): string {
  const timeLabel = type === 'day_before' ? 'tomorrow' : type === 'hour_before' ? 'in 1 hour' : 'in 15 minutes'
  const emoji = type === 'day_before' ? '📅' : type === 'hour_before' ? '⏰' : '🔔'

  const locationLine = formatLocation(data.location)
  const time = `${formatDate(data.startTime)} ${formatTime(data.startTime)} – ${formatTime(data.endTime)}`

  let msg = `${emoji} <b>Event ${timeLabel}!</b>\n\n`
  msg += `<b>${escapeHtml(data.eventTitle)}</b>\n`
  msg += `🏢 ${escapeHtml(data.organizationTitle)}\n`
  msg += `🕐 ${time}\n`
  if (locationLine) msg += `${locationLine}\n`
  if (data.slug) msg += '\nTo cancel, open the event, sign in, and choose Cancel registration.'

  return msg
}

export function getReminderKeyboard(
  eventId: number,
  slug: string,
  chatType: Chat['type'] = 'private',
  baseUrl = env.MAP_APP_URL
): InlineKeyboard {
  const keyboard = new InlineKeyboard().text('✅ Confirm', `reminder:confirm:${eventId}`)

  if (slug) {
    keyboard
      .row()
      .add(createMiniAppButton(baseUrl, chatType, { eventSlug: slug }, '❌ Open event to cancel registration'))
  }

  return keyboard
}
