import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EventPublishReadinessState, EventStatus, LocationType } from '@studyverse/proto'
import {
  buildEventLocation,
  canRequestBooking,
  eventBookingProposalSignature,
  needsPublishedBookingCheck,
  normalizeEventRoomId,
  readinessMessage
} from './event-submission'

test('room aliases use the same canonical ID for comparisons and requests', () => {
  for (const id of ['C1.2.321', ' c1.2.321 ', 'С1.2.321', 'с1.2.321']) {
    assert.equal(normalizeEventRoomId(id), 'C1.2.321')
  }
  for (const id of ['k.101', 'К.101', 'к.101']) assert.equal(normalizeEventRoomId(id), 'K.101')
  assert.equal(normalizeEventRoomId(' atrium '), 'ATRIUM')
  assert.equal(normalizeEventRoomId('334L'), '334L')
})

test('location requests omit hidden fields after format changes', () => {
  const fields = {
    roomId: 'с1.2.321',
    roomName: 'Hall',
    floor: 2,
    externalAddress: 'Campus road',
    googleMapsUrl: 'https://maps.example.test',
    onlineUrl: 'https://meeting.example.test',
    onlinePlatform: 'zoom'
  }
  assert.deepEqual(buildEventLocation({ ...fields, locationType: LocationType.ONLINE }), {
    type: LocationType.ONLINE,
    onlineUrl: fields.onlineUrl,
    onlinePlatform: 'zoom'
  })
  assert.deepEqual(buildEventLocation({ ...fields, locationType: LocationType.CAMPUS_ROOM }), {
    type: LocationType.CAMPUS_ROOM,
    roomId: 'C1.2.321',
    roomName: 'Hall',
    floor: 2
  })
  assert.deepEqual(buildEventLocation({ ...fields, locationType: LocationType.EXTERNAL }), {
    type: LocationType.EXTERNAL,
    externalAddress: fields.externalAddress,
    googleMapsUrl: fields.googleMapsUrl
  })
})

test('published metadata and equivalent room aliases bypass booking checks', () => {
  const current = {
    status: EventStatus.PUBLISHED,
    startTime: '2027-01-01T10:00:00Z',
    endTime: '2027-01-01T11:00:00Z',
    structuredLocation: { type: LocationType.CAMPUS_ROOM, roomId: 'С1.2.321' }
  }
  const proposal = {
    startTime: '2027-01-01T10:00:00.000Z',
    endTime: '2027-01-01T11:00:00.000Z',
    structuredLocation: { type: LocationType.CAMPUS_ROOM, roomId: 'C1.2.321', roomName: 'Renamed hall' }
  }
  assert.equal(needsPublishedBookingCheck(current, proposal), false)
  assert.equal(needsPublishedBookingCheck(current, { ...proposal, startTime: '2027-01-01T10:15:00Z' }), true)
  assert.equal(
    needsPublishedBookingCheck(current, { ...proposal, structuredLocation: { type: LocationType.ONLINE } }),
    true
  )
  assert.equal(
    needsPublishedBookingCheck(current, {
      ...proposal,
      structuredLocation: { type: LocationType.CAMPUS_ROOM, roomId: 'C1.2.322' }
    }),
    true
  )
  assert.equal(needsPublishedBookingCheck({ ...current, status: EventStatus.DRAFT }, proposal), false)
  assert.equal(
    needsPublishedBookingCheck(
      { ...current, structuredLocation: { type: LocationType.ONLINE, roomId: 'stale room' } },
      { ...proposal, structuredLocation: { type: LocationType.ONLINE } }
    ),
    false
  )
})

test('missing, rejected and cancelled bookings can be requested again, pending and approved cannot', () => {
  for (const state of [
    EventPublishReadinessState.ROOM_BOOKING_MISSING,
    EventPublishReadinessState.ROOM_BOOKING_REJECTED,
    EventPublishReadinessState.ROOM_BOOKING_CANCELLED
  ])
    assert.equal(canRequestBooking(state), true)
  for (const state of [
    EventPublishReadinessState.WAITING_ROOM_APPROVAL,
    EventPublishReadinessState.READY,
    EventPublishReadinessState.NOT_DRAFT
  ])
    assert.equal(canRequestBooking(state), false)
  assert.equal(
    readinessMessage({ state: EventPublishReadinessState.WAITING_ROOM_APPROVAL }),
    'Waiting for room approval. Your event has not changed. Save again after approval.'
  )
})

test('proposal signatures follow RHF room, format and calendar changes but preserve equivalent aliases', () => {
  const original = {
    locationType: LocationType.CAMPUS_ROOM,
    roomId: 'C1.2.321',
    hasTime: true,
    startDate: new Date('2027-10-21'),
    endDate: new Date('2027-10-21'),
    startTime: '12:00',
    endTime: '13:00'
  }
  const signature = eventBookingProposalSignature(original)
  assert.equal(eventBookingProposalSignature({ ...original, roomId: 'с1.2.321' }), signature)
  for (const change of [
    { locationType: LocationType.ONLINE },
    { roomId: 'C1.2.322' },
    { hasTime: false },
    { startDate: new Date('2027-10-22') },
    { endDate: new Date('2027-10-22') },
    { startTime: '12:30' },
    { endTime: '13:30' }
  ])
    assert.notEqual(eventBookingProposalSignature({ ...original, ...change }), signature)
})
