import type { MessageInitShape } from '@bufbuild/protobuf'
import { createConnectQueryKey } from '@connectrpc/connect-query'
import {
  type GetEventPublishReadinessRequestSchema,
  getEvent,
  getEventPublishReadiness,
  getEventRegistrations,
  listCategories,
  listEventsForAdmin,
  listOrganizations,
  listRooms
} from '@studyverse/proto'

export function eventsListKey() {
  return createConnectQueryKey({ schema: listEventsForAdmin, cardinality: 'finite' })
}

export function eventDetailKey(eventId: number) {
  return createConnectQueryKey({ schema: getEvent, input: { id: eventId }, cardinality: 'finite' })
}

// Omitting the proposal invalidates every readiness variant for this event.
export function eventPublishReadinessKey(input?: MessageInitShape<typeof GetEventPublishReadinessRequestSchema>) {
  return createConnectQueryKey({ schema: getEventPublishReadiness, input, cardinality: 'finite' })
}

export function categoriesListKey() {
  return createConnectQueryKey({ schema: listCategories, cardinality: 'finite' })
}

export function organizationsListKey() {
  return createConnectQueryKey({ schema: listOrganizations, cardinality: 'finite' })
}

export function roomsListKey() {
  return createConnectQueryKey({ schema: listRooms, cardinality: 'finite' })
}

export function eventRegistrationsKey(eventId: number) {
  return createConnectQueryKey({
    schema: getEventRegistrations,
    input: { eventId },
    cardinality: 'finite'
  })
}
