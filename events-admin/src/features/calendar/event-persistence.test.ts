import assert from 'node:assert/strict'
import { test } from 'node:test'
import { persistCalendarEvent } from './event-persistence'
import type { IEvent } from './interfaces'

const original: IEvent = {
  id: 9,
  title: 'Test event',
  description: '',
  color: 'blue',
  startDate: '2026-10-02T09:00:00.000Z',
  endDate: '2026-10-02T10:00:00.000Z',
  user: { id: '1', name: 'Host', picturePath: null }
}
const moved = { ...original, startDate: '2026-10-03T09:00:00.000Z', endDate: '2026-10-03T10:00:00.000Z' }

test('calendar persists drag and resize dates before confirming success', async () => {
  const states: IEvent[] = []
  const requests: unknown[] = []
  let release = () => {}
  const pending = persistCalendarEvent(
    original,
    moved,
    (event) => states.push(event),
    async (input) => {
      requests.push(input)
      await new Promise<void>((resolve) => {
        release = resolve
      })
    }
  )
  assert.deepEqual(states, [moved])
  assert.deepEqual(requests, [{ id: 9, startTime: moved.startDate, endTime: moved.endDate }])
  release()
  await pending
  assert.deepEqual(states, [moved])
})

test('calendar rolls back optimistic dates when RPC rejects', async () => {
  const states: IEvent[] = []
  await assert.rejects(
    persistCalendarEvent(
      original,
      moved,
      (event) => states.push(event),
      async () => {
        throw new Error('Permission denied')
      }
    ),
    /Permission denied/
  )
  assert.deepEqual(states, [moved, original])
})
