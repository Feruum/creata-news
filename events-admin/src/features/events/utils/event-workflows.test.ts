import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Code, ConnectError } from '@connectrpc/connect'
import { copyPublicEventLink, finishCampusDraft, getSubmitIntent } from './event-workflows'

test('each submission gets its own intent, including retries and Enter submissions', () => {
  assert.equal(getSubmitIntent({ value: 'draft' }), 'draft')
  assert.equal(getSubmitIntent(null), 'publish')
  assert.equal(getSubmitIntent({ value: 'publish' }), 'publish')
  assert.equal(getSubmitIntent({ value: 'draft' }), 'draft')
})

test('campus draft requests booking, checks readiness and publishes only after approval', async () => {
  const calls: string[] = []
  const outcome = await finishCampusDraft({
    requestBooking: async () => {
      calls.push('booking')
    },
    checkReadiness: async () => {
      calls.push('readiness')
      return { canPublishNow: true }
    },
    publish: async () => {
      calls.push('publish')
    }
  })
  assert.equal(outcome, 'published')
  assert.deepEqual(calls, ['booking', 'readiness', 'publish'])
})

test('campus draft survives booking failure and pending approval without publishing', async () => {
  let publishes = 0
  assert.equal(
    await finishCampusDraft({
      requestBooking: async () => {
        throw new Error('Booking failed')
      },
      checkReadiness: async () => ({ canPublishNow: true }),
      publish: async () => {
        publishes++
      }
    }),
    'booking-failed'
  )
  assert.equal(
    await finishCampusDraft({
      requestBooking: async () => {},
      checkReadiness: async () => ({ canPublishNow: false }),
      publish: async () => {
        publishes++
      }
    }),
    'pending'
  )
  assert.equal(publishes, 0)
})

test('an already existing booking refreshes readiness and never creates another event', async () => {
  const calls: string[] = []
  assert.equal(
    await finishCampusDraft({
      requestBooking: async () => {
        calls.push('booking')
        throw new ConnectError('already exists', Code.AlreadyExists)
      },
      checkReadiness: async () => {
        calls.push('readiness')
        return { canPublishNow: false }
      },
      publish: async () => {
        calls.push('publish')
      }
    }),
    'pending'
  )
  assert.deepEqual(calls, ['booking', 'readiness'])
})

test('copy uses the public slug route and waits for the clipboard', async () => {
  let written = ''
  let release = () => {}
  const clipboard = {
    writeText: (text: string) => {
      written = text
      return new Promise<void>((resolve) => {
        release = resolve
      })
    }
  }
  let done = false
  const pending = copyPublicEventLink(
    { id: 42, slug: 'hello world' },
    'https://map.webspace.cat/administration/',
    clipboard
  ).then(() => {
    done = true
  })
  await Promise.resolve()
  assert.equal(written, 'https://map.webspace.cat/event/hello%20world')
  assert.equal(done, false)
  release()
  await pending
  assert.equal(done, true)
})

test('clipboard failures propagate and events without slugs use their public ID route', async () => {
  await assert.rejects(
    copyPublicEventLink({ id: 42, slug: '' }, 'https://map.webspace.cat', {
      writeText: async (url) => {
        assert.equal(url, 'https://map.webspace.cat/event/42')
        throw new Error('Clipboard denied')
      }
    }),
    /Clipboard denied/
  )
})
