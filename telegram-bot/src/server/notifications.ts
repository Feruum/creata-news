import { createHash, timingSafeEqual } from 'node:crypto'
import { AbortController as TelegramAbortController } from 'abort-controller'
import { Elysia, t } from 'elysia'
import type { Bot } from 'grammy'

export const MAX_NOTIFICATION_BODY_BYTES = 2 * 1024 * 1024
export const MAX_NOTIFICATION_MESSAGE_LENGTH = 4096
export const MAX_BULK_NOTIFICATIONS = 100
// Complete before the backend's 30-second HTTP deadline so durable callers can retry.
export const NOTIFICATION_DELIVERY_TIMEOUT_MS = 25_000

const telegramId = t.Union([
  t.Integer({ minimum: 1, maximum: Number.MAX_SAFE_INTEGER }),
  t.Integer({ minimum: Number.MIN_SAFE_INTEGER, maximum: -1 })
])
const message = t.String({ minLength: 1, maxLength: MAX_NOTIFICATION_MESSAGE_LENGTH })
const parseMode = t.Optional(t.Union([t.Literal('HTML'), t.Literal('Markdown'), t.Literal('MarkdownV2')]))
const buttonText = t.String({ minLength: 1, maxLength: 64 })
const replyMarkup = t.Optional(
  t.Object({
    inline_keyboard: t.Array(
      t.Array(
        t.Union([
          t.Object({ text: buttonText, url: t.String({ minLength: 1, maxLength: 2048 }) }),
          t.Object({ text: buttonText, callback_data: t.String({ minLength: 1, maxLength: 64 }) })
        ]),
        { minItems: 1, maxItems: 8 }
      ),
      { minItems: 1, maxItems: 20 }
    )
  })
)
const notifyBody = t.Object({ telegramId, message, parseMode, replyMarkup })
const bulkNotifyBody = t.Object({
  notifications: t.Array(t.Object({ telegramId, message }), { minItems: 1, maxItems: MAX_BULK_NOTIFICATIONS }),
  parseMode,
  replyMarkup
})

function rejection(status: number, error: string): Response {
  return Response.json({ success: false, error }, { status })
}

function deliveryDeadline(request: Request, timeoutMs: number) {
  // Grammy's Node transport uses abort-controller's signal type.
  const controller = new TelegramAbortController()
  const abort = () => controller.abort()
  if (request.signal.aborted) abort()
  else request.signal.addEventListener('abort', abort, { once: true })
  const timer = setTimeout(abort, timeoutMs)
  return {
    signal: controller.signal,
    close: () => {
      clearTimeout(timer)
      request.signal.removeEventListener('abort', abort)
    }
  }
}

export function createNotificationRoutes(
  api: Pick<Bot['api'], 'sendMessage'>,
  secret: string,
  deliveryTimeoutMs = NOTIFICATION_DELIVERY_TIMEOUT_MS
) {
  const expected = createHash('sha256').update(secret).digest()
  const parsedBodies = new WeakMap<Request, unknown>()
  return new Elysia({ name: 'internal-notifications' })
    .onRequest(async ({ request }) => {
      const path = decodeURIComponent(new URL(request.url).pathname).replace(/\/+$/, '')
      if (path !== '/notify' && path !== '/notify/bulk') return
      const candidate = request.headers.get('X-Internal-Secret')
      const received = createHash('sha256')
        .update(candidate ?? '')
        .digest()
      if (!secret || !candidate || !timingSafeEqual(expected, received)) return rejection(401, 'unauthorized')
      if (!request.headers.get('Content-Type')?.toLowerCase().startsWith('application/json')) {
        return rejection(415, 'json_required')
      }
      if (Number(request.headers.get('Content-Length')) > MAX_NOTIFICATION_BODY_BYTES) {
        return rejection(413, 'body_too_large')
      }
      const reader = request.body?.getReader()
      if (!reader) return rejection(400, 'invalid_json')
      const chunks: Uint8Array[] = []
      let total = 0
      try {
        while (true) {
          const chunk = await reader.read()
          if (chunk.done) break
          total += chunk.value.byteLength
          if (total > MAX_NOTIFICATION_BODY_BYTES) {
            await reader.cancel()
            return rejection(413, 'body_too_large')
          }
          chunks.push(chunk.value)
        }
        const body = new Uint8Array(total)
        let offset = 0
        for (const chunk of chunks) {
          body.set(chunk, offset)
          offset += chunk.byteLength
        }
        parsedBodies.set(request, JSON.parse(new TextDecoder('utf-8', { fatal: true }).decode(body)))
      } catch {
        return rejection(400, 'invalid_json')
      } finally {
        reader.releaseLock()
      }
    })
    .onParse(({ request }) => parsedBodies.get(request))
    .post(
      '/notify',
      async ({ body, set, request }) => {
        const deadline = deliveryDeadline(request, deliveryTimeoutMs)
        try {
          await api.sendMessage(
            body.telegramId,
            body.message,
            { parse_mode: body.parseMode, reply_markup: body.replyMarkup },
            deadline.signal
          )
          return { success: true }
        } catch {
          set.status = 502
          return { success: false, error: 'delivery_failed' }
        } finally {
          deadline.close()
        }
      },
      { body: notifyBody }
    )
    .post(
      '/notify/bulk',
      async ({ body, set, request }) => {
        const deadline = deliveryDeadline(request, deliveryTimeoutMs)
        try {
          const results = await Promise.allSettled(
            body.notifications.map(({ telegramId, message }) =>
              api.sendMessage(
                telegramId,
                message,
                { parse_mode: body.parseMode, reply_markup: body.replyMarkup },
                deadline.signal
              )
            )
          )
          const sent = results.filter((result) => result.status === 'fulfilled').length
          const failed = results.length - sent
          if (failed > 0) set.status = 502
          return { success: failed === 0, sent, failed, total: results.length }
        } finally {
          deadline.close()
        }
      },
      { body: bulkNotifyBody }
    )
}
