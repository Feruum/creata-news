import { createClient } from '@connectrpc/connect'
import { EventRegistrationsService, EventsService } from '@studyverse/proto'
import { env } from './env'
import { createBackendTransport } from './rpc-transport'

const transport = createBackendTransport({
  baseUrl: env.EMS_API_URL,
  internalReminderSecret: env.INTERNAL_REMINDER_SECRET
})

export const eventsClient = createClient(EventsService, transport)
export const registrationsClient = createClient(EventRegistrationsService, transport)
