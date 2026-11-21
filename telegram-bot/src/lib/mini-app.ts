import { InlineKeyboard } from 'grammy'
import type { Chat, InlineKeyboardButton, MenuButton } from 'grammy/types'
import { z } from 'zod'

export const DEFAULT_MAP_APP_URL = 'https://map.webspace.cat'

export const mapAppUrlSchema = z
  .string()
  .url()
  .refine((value) => /^https?:\/\//i.test(value), 'MAP_APP_URL must use HTTP or HTTPS')

export type MiniAppTarget = { roomId: string } | { eventSlug: string }
type ChatType = Chat['type'] | undefined

export function generateMiniAppUrl(baseUrl: string, target?: MiniAppTarget): string {
  const url = new URL(baseUrl)
  if (target) {
    const id = 'roomId' in target ? target.roomId : target.eventSlug
    if (id) {
      const route = 'roomId' in target ? 'room' : 'event'
      url.pathname = `${url.pathname.replace(/\/+$/, '')}/${route}/${encodeURIComponent(id)}`
      url.search = ''
      url.hash = ''
    }
  }
  return url.href.replace(/\/$/, '')
}

export function createMiniAppButton(
  baseUrl: string,
  chatType: ChatType,
  target?: MiniAppTarget,
  text = 'Открыть карту'
): InlineKeyboardButton {
  const url = generateMiniAppUrl(baseUrl, target)
  return chatType === 'private' && new URL(url).protocol === 'https:'
    ? InlineKeyboard.webApp(text, url)
    : InlineKeyboard.url(text, url)
}

export function getMiniAppMenuButton(baseUrl: string): MenuButton {
  const url = generateMiniAppUrl(baseUrl)
  return new URL(url).protocol === 'https:'
    ? { type: 'web_app', text: 'Открыть карту', web_app: { url } }
    : { type: 'commands' }
}
