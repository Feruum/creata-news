import { spyOn } from 'bun:test'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { EventStatus, LocationType, type Event as ProtoEvent } from '@studyverse/proto'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { fetchUpcomingEvents } = await import('./events')

test('upcoming events scan ID-ordered RPC pages, sort by time and paginate only future published events', async () => {
  const now = Date.parse('2026-10-02T00:00:00Z')
  const event = (id: number, startTime: string, status = EventStatus.PUBLISHED): ProtoEvent => ({
    $typeName: 'events.v1.Event',
    id,
    title: `Event ${id}`,
    description: 'Description',
    startTime,
    endTime: '2026-10-04T01:00:00Z',
    slug: `event-${id}`,
    status,
    userId: 1,
    organizationId: 1,
    format: 0,
    createdAt: '',
    updatedAt: '',
    totalRegistrations: 0,
    totalAttendees: 0,
    publishOnMap: true,
    publishOnSundayOpen: false,
    externalRegistrationCount: 0,
    allocatedSeatCount: 0,
    collaborators: [],
    organizers: [],
    checkInPhase: 0,
    organization: {
      $typeName: 'events.v1.Organization',
      id: 1,
      title: 'Study club',
      organizationTypeId: 1,
      status: 0,
      createdAt: '',
      updatedAt: '',
      slug: 'club'
    },
    structuredLocation: { $typeName: 'events.v1.EventLocation', type: LocationType.CAMPUS_ROOM, roomName: 'C1.301' }
  })
  const pages = [
    [event(1, '2026-10-01T00:00:00Z'), event(2, '2026-10-04T00:00:00Z')],
    [event(3, '2026-10-03T00:00:00Z'), event(4, '2026-10-03T00:00:00Z')],
    [event(5, 'invalid'), event(6, '2026-10-03T00:00:00Z', EventStatus.CANCELLED)]
  ]
  const requested: number[] = []
  const noNetwork = spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Legacy REST calls are not supported'))
  try {
    const result = await fetchUpcomingEvents(1, 1, {
      now: () => now,
      client: {
        listEvents: async (request) => {
          requested.push(request.page || 0)
          assert.equal(request.status, EventStatus.PUBLISHED)
          return { $typeName: 'events.v1.ListEventsResponse', events: pages[(request.page || 1) - 1] || [], total: 6 }
        }
      }
    })
    assert.deepEqual(requested, [1, 2, 3])
    assert.equal(result.total, 3)
    assert.deepEqual(
      result.events.map((value) => value.id),
      ['4']
    )
    assert.equal(result.events[0].slug, 'event-4')
    assert.equal(result.events[0].startDateTime, '2026-10-03T00:00:00Z')
    assert.equal(result.events[0].location, 'C1.301')
    assert.equal(result.events[0].organization, 'Study club')
  } finally {
    noNetwork.mockRestore()
  }
})
