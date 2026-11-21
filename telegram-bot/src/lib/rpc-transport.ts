import { createConnectTransport } from '@connectrpc/connect-web'

export function createBackendTransport(options: {
  baseUrl: string
  internalReminderSecret: string
  fetch?: typeof globalThis.fetch
}) {
  return createConnectTransport({
    baseUrl: options.baseUrl,
    fetch: options.fetch,
    interceptors: [(next) => async (request) => {
      if (
        request.service.typeName === 'events.v1.EventsService' &&
        ['GetPendingReminders', 'MarkRemindersSent'].includes(request.method.name)
      ) {
        request.header.set('X-Internal-Secret', options.internalReminderSecret)
      }
      return next(request)
    }]
  })
}
