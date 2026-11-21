import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isEventCalendarDateDisabled } from './event-date-helpers'

test('event calendar allows today after midnight and rejects earlier days', () => {
  const now = new Date(2026, 9, 2, 1, 49)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 2), undefined, now), false)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 1), undefined, now), true)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 3), undefined, now), false)
})

test('end calendar compares calendar days instead of the start time', () => {
  const now = new Date(2026, 9, 2, 1, 49)
  const start = new Date(2026, 9, 3, 12, 30)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 3), start, now), false)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 2), start, now), true)
  assert.equal(isEventCalendarDateDisabled(new Date(2026, 9, 4), start, now), false)
})
