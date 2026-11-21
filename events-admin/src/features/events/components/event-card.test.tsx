import assert from 'node:assert/strict'
import { test } from 'node:test'
import { type Event, EventStatus } from '@studyverse/proto'
import { renderToStaticMarkup } from 'react-dom/server'
import { SimpleEventCard } from './event-card'

test('mobile cards distinguish lifecycle status from format even when dates suggest otherwise', () => {
  for (const [status, label] of [
    [EventStatus.DRAFT, 'Draft'],
    [EventStatus.CANCELLED, 'Cancelled']
  ] as const) {
    const event = {
      title: 'QA event',
      status,
      startTime: '2020-01-01T10:00:00Z',
      endTime: '2020-01-01T11:00:00Z'
    } as Event
    assert.ok(renderToStaticMarkup(<SimpleEventCard event={event} />).includes(label))
  }
})
