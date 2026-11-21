import assert from 'node:assert/strict'
import { test } from 'node:test'
import { create } from '@bufbuild/protobuf'
import { EventBadgeStatus, EventSchema } from '@studyverse/proto'
import { renderToStaticMarkup } from 'react-dom/server'
import { EventBadgeReview } from './event-badge-review'
import { EventBadges } from './event-badges'

test('only approved event badges are shown as confirmed', () => {
  const event = create(EventSchema, {
    socGpaBadge: { status: EventBadgeStatus.PENDING },
    irosBadge: { status: EventBadgeStatus.REJECTED }
  })
  assert.equal(renderToStaticMarkup(<EventBadges event={event} />), '')
  event.socGpaBadge = { $typeName: 'events.v1.EventBadge', status: EventBadgeStatus.APPROVED }
  const html = renderToStaticMarkup(<EventBadges event={event} />)
  assert.ok(html.includes('SocGpa'))
  assert.ok(!html.includes('iRos'))
})

test('management views distinguish requests from approved badges', () => {
  const event = create(EventSchema, {
    socGpaBadge: { status: EventBadgeStatus.PENDING },
    irosBadge: { status: EventBadgeStatus.REJECTED }
  })
  const html = renderToStaticMarkup(<EventBadges event={event} showRequests />)
  assert.ok(html.includes('Pending approval'))
  assert.ok(html.includes('Rejected'))
})

test('event editors can see requests without administrator review controls', () => {
  const event = create(EventSchema, { socGpaBadge: { status: EventBadgeStatus.PENDING } })
  const html = renderToStaticMarkup(<EventBadgeReview event={event} canReview={false} onReview={() => {}} />)
  assert.ok(html.includes('Pending approval'))
  assert.ok(!html.includes('<button'))
})

test('administrators can review each requested badge independently', () => {
  const event = create(EventSchema, {
    socGpaBadge: { status: EventBadgeStatus.PENDING },
    irosBadge: { status: EventBadgeStatus.APPROVED }
  })
  const html = renderToStaticMarkup(<EventBadgeReview event={event} canReview onReview={() => {}} />)
  assert.ok(html.includes('Approve SocGpa'))
  assert.ok(html.includes('Reject SocGpa'))
  assert.ok(html.includes('Remove iRos approval'))
  assert.ok(!html.includes('Approve iRos'))
})
