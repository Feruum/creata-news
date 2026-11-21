import assert from 'node:assert/strict'
import { test } from 'node:test'
import { mutationCategory, pageCategory } from './product-events'

test('identifies write mutations from real Connect key descriptors without using request input', () => {
  assert.equal(
    mutationCategory([
      'connect-query',
      { serviceName: 'events.v1.EventsService', methodName: 'CreateEvent', input: { title: 'private event title' } }
    ]),
    'event'
  )
  assert.equal(
    mutationCategory(['connect-query', { serviceName: 'users.v1.UsersService', methodName: 'ListUsers' }]),
    null
  )
  assert.equal(
    mutationCategory(['connect-query', { serviceName: 'events.v1.BookingService', methodName: 'ApproveBooking' }]),
    'booking'
  )
})

test('page category is bounded across production base paths and entity URLs', () => {
  assert.equal(pageCategory('/administration/events/42?title=private'), 'event')
  assert.equal(pageCategory('/room/C1.1.101'), 'room')
  assert.equal(pageCategory('/unrecognized/private-path'), 'other')
})
