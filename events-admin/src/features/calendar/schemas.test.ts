import assert from 'node:assert/strict'
import { test } from 'node:test'
import { zodResolver } from '@hookform/resolvers/zod'
import { eventSchema } from './schemas'

test('calendar form preserves required date and variant messages', () => {
  const result = eventSchema.safeParse({
    title: 'Test event',
    description: 'Event description',
    startTime: '12:00',
    endTime: '13:00'
  })

  assert.equal(result.success, false)
  if (result.success) throw new Error('Missing dates and variant must be rejected')
  assert.deepEqual(
    result.error.issues.map(({ path, message }) => ({ path, message })),
    [
      { path: ['startDate'], message: 'Start date is required' },
      { path: ['endDate'], message: 'End date is required' },
      { path: ['color'], message: 'Variant is required' }
    ]
  )
})

test('calendar resolver submits parsed defaults and returns field errors for invalid times', async () => {
  const input = {
    title: 'Test event',
    description: 'Event description',
    startDate: new Date('2027-10-21'),
    endDate: new Date('2027-10-21'),
    startTime: '12:00',
    endTime: '13:00',
    color: 'blue' as const
  }
  const resolve = zodResolver(eventSchema)
  const options = { fields: {}, shouldUseNativeValidation: false }

  assert.deepEqual(await resolve(input, undefined, options), {
    errors: {},
    values: { ...input, hasTime: true }
  })

  const missingTimes = await resolve({ ...input, startTime: '', endTime: '' }, undefined, options)
  assert.equal(missingTimes.errors.startTime?.message, 'Start time is required')
  assert.equal(missingTimes.errors.endTime?.message, 'End time is required')
  assert.deepEqual(missingTimes.values, {})

  const reversedTimes = await resolve({ ...input, endTime: '11:00' }, undefined, options)
  assert.equal(reversedTimes.errors.endDate?.message, 'End datetime must be after start')
  assert.deepEqual(reversedTimes.values, {})
})
