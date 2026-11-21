import assert from 'node:assert/strict'
import { test } from 'node:test'
import { CheckInPhase, EventStatus } from '@studyverse/proto'
import { getEventStatus } from './event-status'

test('lifecycle statuses take precedence over past and future event dates', () => {
  for (const [status, expected] of [
    [EventStatus.DRAFT, 'draft'],
    [EventStatus.CANCELLED, 'cancelled'],
    [EventStatus.ARCHIVED, 'archived']
  ] as const) {
    for (const year of [2020, 2099]) {
      const event = {
        status,
        checkInPhase: CheckInPhase.UNSPECIFIED,
        startTime: `${year}-01-01T09:00:00Z`,
        endTime: `${year}-01-01T10:00:00Z`
      }
      assert.equal(getEventStatus(event), expected)
    }
  }
})
