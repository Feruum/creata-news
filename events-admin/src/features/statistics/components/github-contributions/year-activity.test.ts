import assert from 'node:assert/strict'
import { test } from 'node:test'
import { fillActivityYear } from './year-activity'

test('fills all dates of sparse and empty years, including leap day', () => {
  const sparse = fillActivityYear([{ date: '2026-10-28', count: 9, level: 4 }], 2026)
  assert.equal(sparse.length, 365)
  assert.deepEqual(sparse[0], { date: '2026-01-01', count: 0, level: 0 })
  assert.equal(sparse.at(-1)?.date, '2026-12-31')
  assert.equal(
    sparse.reduce((sum, day) => sum + day.count, 0),
    9
  )
  const leap = fillActivityYear([], 2024)
  assert.equal(leap.length, 366)
  assert.equal(
    leap.some((day) => day.date === '2024-02-29'),
    true
  )
  assert.equal(
    fillActivityYear(sparse, 2025).every((day) => day.count === 0),
    true
  )
})
