import assert from 'node:assert/strict'
import { test } from 'node:test'
import { campusDate, isLessonReminderDue } from './lesson-time'

test('15-minute reminder uses the lesson date and timezone, never repeats a week later', () => {
  const start = '2026-09-28T10:00:00+05:00'
  assert.equal(isLessonReminderDue(start, Date.parse('2026-09-28T04:45:00Z')), true)
  assert.equal(isLessonReminderDue(start, Date.parse('2026-09-28T04:46:00Z')), true)
  assert.equal(isLessonReminderDue(start, Date.parse('2026-10-05T04:45:00Z')), false)
  assert.equal(isLessonReminderDue('invalid'), false)
  assert.equal(campusDate(new Date('2026-09-27T20:00:00Z')), '2026-09-28')
})

test('lesson retry window ends one minute before the start and never starts early', () => {
  const start = '2026-10-02T05:00:00Z'
  assert.equal(isLessonReminderDue(start, Date.parse('2026-10-02T04:44:59Z')), false)
  assert.equal(isLessonReminderDue(start, Date.parse('2026-10-02T04:58:59Z')), true)
  assert.equal(isLessonReminderDue(start, Date.parse('2026-10-02T04:59:00Z')), false)
  assert.equal(isLessonReminderDue(start, Date.parse('2026-10-02T05:00:00Z')), false)
})
