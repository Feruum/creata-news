import assert from 'node:assert/strict'
import { test } from 'node:test'
import { create } from '@bufbuild/protobuf'
import { GetEventPublishReadinessResponseSchema, LocationType } from '@studyverse/proto'
import { QueryClient } from '@tanstack/react-query'
import { eventPublishReadinessKey } from './keys'

test('readiness cache separates proposed room and times and invalidates every event variant', async () => {
  const client = new QueryClient()
  const original = eventPublishReadinessKey({ eventId: 9 })
  const proposed = {
    eventId: 9,
    proposedStartTime: '2027-01-01T10:00:00Z',
    proposedEndTime: '2027-01-01T11:00:00Z',
    proposedLocation: { type: LocationType.CAMPUS_ROOM, roomId: 'C1.2.321' }
  }
  const variants = [
    original,
    eventPublishReadinessKey(proposed),
    eventPublishReadinessKey({ ...proposed, proposedStartTime: '2027-01-01T10:30:00Z' }),
    eventPublishReadinessKey({ ...proposed, proposedEndTime: '2027-01-01T11:30:00Z' }),
    eventPublishReadinessKey({ ...proposed, proposedLocation: { ...proposed.proposedLocation, roomId: 'C1.2.322' } })
  ]
  for (const [index, key] of variants.entries()) {
    client.setQueryData(key, create(GetEventPublishReadinessResponseSchema, { bookingRequestId: index }))
  }
  const otherEvent = eventPublishReadinessKey({ eventId: 10 })
  client.setQueryData(otherEvent, create(GetEventPublishReadinessResponseSchema))
  assert.equal(client.getQueryCache().getAll().length, 6)
  await client.invalidateQueries({ queryKey: original })
  for (const key of variants) assert.equal(client.getQueryState(key)?.isInvalidated, true)
  assert.equal(client.getQueryState(otherEvent)?.isInvalidated, false)
  client.clear()
})
