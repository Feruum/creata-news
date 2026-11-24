import { expect, test } from 'bun:test'
import { ApiError } from '@creata/contracts'
import { acknowledgeStoryUpdates, waitForStoryReads } from './story-reading'

test('retries a failed acknowledgement and keeps the list waiting for the same reader', async () => {
  let attempts = 0
  let release = () => {}
  const finalAttempt = new Promise<void>(resolve => { release = resolve })
  const saved = acknowledgeStoryUpdates('reader-a', async () => {
    if (++attempts === 1) throw new ApiError(503, 'Unavailable')
    await finalAttempt
  })
  let listReady = false
  const list = waitForStoryReads('reader-a').then(() => { listReady = true })
  await waitForStoryReads('reader-b')
  expect(listReady).toBe(false)
  release()
  expect(await saved).toBe(true)
  await list
  expect(listReady).toBe(true)
  expect(attempts).toBe(2)
})

test('does not retry a permanent rejection', async () => {
  let attempts = 0
  const saved = await acknowledgeStoryUpdates('reader-c', async () => {
    attempts++
    throw new ApiError(403, 'Forbidden')
  })
  expect(saved).toBe(false)
  expect(attempts).toBe(1)
})
