import { timingSafeEqual } from 'node:crypto'
import { Elysia, t } from 'elysia'
import type { Update } from 'grammy/types'
import { bot } from '../bot'
import { buildInfo } from '../lib/build-info'
import { env } from '../lib/env'
import { emojiRoutes } from './emoji'
import { createNotificationRoutes } from './notifications'

let webhookReady = false

function validWebhookSecret(candidate: string | undefined): boolean {
  if (!candidate) return false
  const expected = Buffer.from(env.TELEGRAM_WEBHOOK_SECRET)
  const received = Buffer.from(candidate)
  return expected.length === received.length && timingSafeEqual(Uint8Array.from(expected), Uint8Array.from(received))
}

export const server = new Elysia()
  .use(emojiRoutes)
  .post(
    '/telegram/webhook',
    async ({ body, headers, set }) => {
      if (!validWebhookSecret(headers['x-telegram-bot-api-secret-token'])) {
        set.status = 401
        return { success: false, error: 'invalid_webhook_secret' }
      }
      await bot.handleUpdate(body as Update)
      return { success: true }
    },
    { body: t.Unknown() }
  )
  .use(createNotificationRoutes(bot.api, env.INTERNAL_REMINDER_SECRET))
  .get('/health', () => ({ status: 'ok', bot: 'running' }))
  .get('/health/live', () => ({ status: 'ok' }))
  .get('/health/ready', ({ set }) => {
    if (!webhookReady) {
      set.status = 503
      return { status: 'not_ready', webhook: false }
    }
    return { status: 'ready', webhook: true }
  })
  .get('/version', () => buildInfo)

export function startServer() {
  const port = env.HTTP_PORT
  server.listen(port)
  console.log(`HTTP server listening on port ${port}`)
}

export function markWebhookReady() {
  webhookReady = true
}
