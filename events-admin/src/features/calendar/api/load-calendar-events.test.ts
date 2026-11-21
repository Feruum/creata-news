import assert from 'node:assert/strict'
import { test } from 'node:test'
import { loadCalendarEvents } from './load-calendar-events'

test('calendar keeps events beyond the API page cap', async () => {
  const source = Array.from({ length: 451 }, (_, id) => ({ id }))
  const pages: number[] = []
  const result = await loadCalendarEvents(async ({ page, limit }) => {
    assert.equal(limit, 200)
    pages.push(page)
    return { events: source.slice((page - 1) * limit, page * limit), total: source.length }
  })
  assert.deepEqual(result.events, source)
  assert.deepEqual(pages, [1, 2, 3])
})

test('calendar stops if a changing dataset returns an empty page', async () => {
  let calls = 0
  const result = await loadCalendarEvents(async () => {
    calls += 1
    return { events: [], total: 500 }
  })
  assert.deepEqual(result.events, [])
  assert.equal(calls, 1)
})
