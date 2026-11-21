import assert from 'node:assert/strict'
import { test } from 'node:test'
import { zodResolver } from '@hookform/resolvers/zod'
import { EventFormat, LocationType } from '@studyverse/proto'
import { createEventSchema, defaultValues, fullEventSchema, scheduleSchema } from './schema'

test('event and schedule forms preserve required date messages', () => {
  for (const schema of [createEventSchema, scheduleSchema]) {
    const result = schema.safeParse({
      title: 'Test event',
      description: 'A long enough event description',
      organizationIds: [1],
      roomName: 'Lecture hall',
      startTime: '12:00',
      endTime: '13:00'
    })

    assert.equal(result.success, false)
    if (result.success) throw new Error('Missing dates must be rejected')
    assert.deepEqual(
      result.error.issues.map(({ path, message }) => ({ path, message })),
      [
        { path: ['startDate'], message: 'Start date is required' },
        { path: ['endDate'], message: 'End date is required' }
      ]
    )
  }
})

test('event resolver supplies parsed defaults for submitted form values', async () => {
  const input = {
    title: 'Test event',
    description: 'A long enough event description',
    startDate: new Date('2027-10-21'),
    endDate: new Date('2027-10-21'),
    startTime: '12:00',
    endTime: '13:00',
    organizationIds: [1],
    roomName: 'Lecture hall'
  }
  const result = await zodResolver(createEventSchema)(input, undefined, {
    fields: {},
    shouldUseNativeValidation: false
  })

  assert.deepEqual(result, {
    errors: {},
    values: {
      ...input,
      hasTime: true,
      locationType: LocationType.CAMPUS_ROOM,
      requestRoomBooking: true,
      socGpaRequested: false,
      irosRequested: false
    }
  })
})

test('multi-step event resolver coerces capacity and supplies parsed defaults', async () => {
  const input = {
    title: 'Test event',
    description: 'A long enough event description',
    format: EventFormat.OFFLINE,
    startDate: new Date('2027-10-21'),
    endDate: new Date('2027-10-21'),
    startTime: '12:00',
    endTime: '13:00',
    capacity: '42'
  }
  const result = await zodResolver(fullEventSchema)(input, undefined, {
    fields: {},
    shouldUseNativeValidation: false
  })

  assert.deepEqual(result, {
    errors: {},
    values: { ...input, capacity: 42, hasTime: true, isPublic: true }
  })
})

test('publication targets are excluded from event form values and defaults', () => {
  const result = createEventSchema.parse({
    title: 'Host publication workshop',
    description: JSON.stringify([{ type: 'p', children: [{ text: 'A workshop with automatic publication' }] }]),
    startDate: new Date('2027-10-21'),
    endDate: new Date('2027-10-21'),
    startTime: '12:00',
    endTime: '13:00',
    organizationIds: [1],
    locationType: LocationType.ONLINE,
    onlineUrl: 'https://example.test/workshop',
    publishOnMap: true,
    publishOnSundayOpen: true
  })

  for (const field of ['publishOnMap', 'publishOnSundayOpen']) {
    assert.equal(Object.hasOwn(result, field), false)
    assert.equal(Object.hasOwn(defaultValues, field), false)
  }
})

test('event form accepts the JSON description saved by its editor', () => {
  const description = JSON.stringify([{ type: 'p', children: [{ text: 'Prelaunch test draft for Campus Club.' }] }])
  const result = createEventSchema.safeParse({
    title: 'Campus Club prelaunch flow test',
    description,
    startDate: new Date('2026-10-05T00:00:00+05:00'),
    endDate: new Date('2026-10-05T00:00:00+05:00'),
    startTime: '12:00',
    endTime: '13:00',
    organizationIds: [1],
    locationType: LocationType.ONLINE,
    onlineUrl: 'https://example.com'
  })

  assert.equal(result.success, true, result.success ? undefined : JSON.stringify(result.error.flatten()))
})

test('event form rejects multiple primary hosting organizations instead of silently dropping them', () => {
  const result = createEventSchema.safeParse({
    title: 'Test event',
    description: JSON.stringify([{ children: [{ text: 'A long enough description' }] }]),
    startDate: new Date('2026-10-05'),
    endDate: new Date('2026-10-05'),
    startTime: '12:00',
    endTime: '13:00',
    organizationIds: [1, 2],
    locationType: LocationType.ONLINE,
    onlineUrl: 'https://example.com'
  })
  assert.equal(result.success, false)
})

test('editing a map or legacy event preserves its valid description without reopening the editor', () => {
  const base = {
    title: 'Existing map event',
    startDate: new Date('2026-10-05'),
    endDate: new Date('2026-10-05'),
    startTime: '12:00',
    endTime: '13:00',
    organizationIds: [1],
    locationType: LocationType.ONLINE,
    onlineUrl: 'https://example.com'
  }
  for (const description of ['A valid legacy description', JSON.stringify('A valid map description')]) {
    const result = createEventSchema.safeParse({ ...base, description })
    assert.equal(result.success, true)
    if (result.success) assert.equal(result.data.description, description)
  }
  for (const description of [
    '',
    'short',
    JSON.stringify('short'),
    'null',
    '{}',
    '[]',
    '[null]',
    '[{"children":null}]'
  ]) {
    assert.equal(createEventSchema.safeParse({ ...base, description }).success, false)
  }
})

const validEvent = {
  title: 'Valid event',
  description: 'A complete event description',
  startDate: new Date('2027-10-21'),
  endDate: new Date('2027-10-21'),
  startTime: '12:00',
  endTime: '13:00',
  organizationIds: [1],
  locationType: LocationType.ONLINE,
  onlineUrl: 'https://example.test/meeting'
}

test('event validation rejects blank titles, invalid clock times and unsafe visible URLs', () => {
  for (const changes of [
    { title: '    ' },
    { startTime: '25:00' },
    { endTime: '12:75' },
    { startTime: 'noon' },
    { onlineUrl: 'javascript:alert(1)' },
    { onlineUrl: 'ftp://example.test' },
    { organizationIds: [0] }
  ])
    assert.equal(createEventSchema.safeParse({ ...validEvent, ...changes }).success, false, JSON.stringify(changes))
})

test('event validation ignores stale URLs hidden by the chosen format', () => {
  assert.equal(
    createEventSchema.safeParse({
      ...validEvent,
      locationType: LocationType.CAMPUS_ROOM,
      roomId: 'C1.2.321',
      onlineUrl: 'stale invalid URL',
      googleMapsUrl: 'stale invalid URL'
    }).success,
    true
  )
})

test('event validation rejects empty, oversized and unsupported images before any upload', () => {
  for (const coverImage of [
    new File([], 'empty.png', { type: 'image/png' }),
    new File(['x'], 'not-image.txt', { type: 'text/plain' }),
    new File([new Uint8Array(5 * 1024 * 1024 + 1)], 'big.png', { type: 'image/png' })
  ])
    assert.equal(createEventSchema.safeParse({ ...validEvent, coverImage }).success, false)
  assert.equal(
    createEventSchema.safeParse({ ...validEvent, coverImage: new File(['png'], 'cover.png', { type: 'image/png' }) })
      .success,
    true
  )
})
