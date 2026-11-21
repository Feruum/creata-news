import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { clearPrivateSnapshots, getOfflineState, verifyOfflineIdentity } from '@/shared/lib/offline-runtime'
import { getSubmission, listSubmissions, reviewSubmission, ScheduleApiError } from './client'

const originalFetch = globalThis.fetch
const summary = {
  id: 'submission-1',
  groupName: 'SE-2401',
  authorId: 'author-1',
  contentHash: 'hash',
  status: 'pending',
  createdAt: '2026-10-04T05:00:00.000Z',
  reviewedAt: null,
  reviewerId: null,
  rejectionReason: null
}

afterEach(async () => {
  globalThis.fetch = originalFetch
  await clearPrivateSnapshots()
})

test('lists submissions from the root API with credentials, no-store, and encoded cursor/group', async () => {
  let requestedUrl = ''
  let requestedInit: RequestInit | undefined
  globalThis.fetch = async (input, init) => {
    requestedUrl = String(input)
    requestedInit = init
    return Response.json({ submissions: [summary], nextCursor: 'next+cursor' })
  }
  const result = await listSubmissions({ status: 'all', group: ' SE & 2401 ' }, 'cursor+one')
  const url = new URL(requestedUrl, 'https://map.example.test/administration/')
  assert.equal(url.pathname, '/api/schedule/admin/submissions')
  assert.equal(url.searchParams.get('status'), 'all')
  assert.equal(url.searchParams.get('group'), 'SE & 2401')
  assert.equal(url.searchParams.get('cursor'), 'cursor+one')
  assert.equal(requestedInit?.credentials, 'include')
  assert.equal(requestedInit?.cache, 'no-store')
  assert.equal(result.nextCursor, 'next+cursor')
})

test('keeps exact lesson dates and full details from the selected snapshot', async () => {
  const settings = { studyYear: 2026, term: 1, firstWeekStart: '2026-09-07' }
  const lesson = {
    id: 'lesson',
    title: 'Calculus',
    start: '2026-10-05T09:00:00+05:00',
    end: '2026-10-05T10:00:00+05:00',
    teacher: 'Teacher',
    lessonType: 'Lecture',
    location: 'C1.101',
    online: false
  }
  globalThis.fetch = async (input) => {
    assert.equal(new URL(String(input), 'https://map.example.test').searchParams.get('id'), 'submission-1')
    return Response.json({
      submission: { ...summary, settings, events: [lesson] },
      published: null,
      diff: { added: [lesson], removed: [] }
    })
  }
  const result = await getSubmission('submission-1')
  assert.equal(result.submission.events[0].start, lesson.start)
  assert.deepEqual(result.diff.added, [lesson])
})

test('rejects blank reasons before sending a decision', async () => {
  let requestCount = 0
  globalThis.fetch = async () => {
    requestCount += 1
    return Response.json(summary)
  }
  await assert.rejects(reviewSubmission({ id: summary.id, decision: 'reject', reason: '   ' }), /reason/)
  assert.equal(requestCount, 0)
})

test('sends a trimmed rejection reason and accepts the reviewed summary', async () => {
  let body: unknown
  globalThis.fetch = async (_input, init) => {
    body = JSON.parse(String(init?.body))
    assert.equal(init?.method, 'POST')
    assert.equal(init?.cache, 'no-store')
    assert.equal(init?.credentials, 'include')
    return Response.json({ ...summary, status: 'rejected', rejectionReason: 'Incorrect room' })
  }
  await reviewSubmission({ id: summary.id, decision: 'reject', reason: '  Incorrect room  ' })
  assert.deepEqual(body, { id: summary.id, decision: 'reject', reason: 'Incorrect room' })
})

test('preserves conflict status and avoids exposing raw server error details', async () => {
  globalThis.fetch = async () => Response.json({ error: 'Internal storage details' }, { status: 409 })
  await assert.rejects(reviewSubmission({ id: summary.id, decision: 'approve' }), (error: unknown) => {
    assert.ok(error instanceof ScheduleApiError)
    assert.equal(error.status, 409)
    assert.match(error.message, /already been reviewed/)
    assert.doesNotMatch(error.message, /storage/)
    return true
  })
})

test('malformed success responses cannot become reviewable schedule data', async () => {
  globalThis.fetch = async () => Response.json({ submissions: [{ ...summary, status: 'unknown' }], nextCursor: null })
  await assert.rejects(listSubmissions({ status: 'pending' }), /could not be loaded/)
})

test('schedule data from A cannot be returned after B verifies while response parsing is pending', async () => {
  let release: ((data: unknown) => void) | undefined
  let requested: (() => void) | undefined
  const started = new Promise<void>((resolve) => {
    requested = resolve
  })
  const body = new Promise<unknown>((resolve) => {
    release = resolve
  })
  const identity = (id: string) => ({
    identityId: id,
    displayName: id,
    email: `${id}@example.test`,
    verifiedAt: Date.now(),
    sessionExpiresAt: Date.now() + 60_000
  })
  globalThis.fetch = async () => {
    const response = Response.json({})
    response.json = async () => {
      requested?.()
      return body
    }
    return response
  }
  await verifyOfflineIdentity(identity('a'))
  const oldRequest = listSubmissions({ status: 'pending' })
  await started
  await verifyOfflineIdentity(identity('b'))
  release?.({ submissions: [summary], nextCursor: null })
  await assert.rejects(oldRequest, /account changed/)
  assert.equal(getOfflineState().identity?.identityId, 'b')
})
