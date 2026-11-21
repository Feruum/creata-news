import { spyOn } from 'bun:test'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'
import type { ScheduleItem } from './schedule'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { createLessonNotification, checkAndNotify, buildLessonClaim } = await import('./scheduler')
const { db } = await import('../db')

const lesson: ScheduleItem = {
  subject: 'Test subject',
  tutor: 'Test tutor',
  classtimeDay: 'Monday',
  classtimeTime: '09:00–09:50',
  lessonType: 'Lecture',
  room: 'C1.2.301',
  mapRoomId: 'C1.2.301',
  group: 'Test',
  teamsMeetingUrl: ''
}

test('lesson notification opens its mapped room as a private Mini App', () => {
  const notification = createLessonNotification(lesson, 'https://map.webspace.cat')
  assert.match(notification.text, /Test subject/)
  assert.match(notification.text, /https:\/\/map.webspace.cat\/room\/C1.2.301/)
  assert.deepEqual(notification.reply_markup.inline_keyboard, [
    [
      {
        text: 'Открыть карту',
        web_app: { url: 'https://map.webspace.cat/room/C1.2.301' }
      }
    ]
  ])
})

test('unmapped lesson notification opens the map home page', () => {
  const notification = createLessonNotification({ ...lesson, mapRoomId: null }, 'https://map.webspace.cat')
  assert.deepEqual(notification.reply_markup.inline_keyboard, [
    [
      {
        text: 'Открыть карту',
        web_app: { url: 'https://map.webspace.cat' }
      }
    ]
  ])
})

test('HTTP lesson notifications keep ordinary links', () => {
  const notification = createLessonNotification(lesson, 'http://localhost:3000')
  assert.deepEqual(notification.reply_markup.inline_keyboard, [
    [
      {
        text: 'Открыть карту',
        url: 'http://localhost:3000/room/C1.2.301'
      }
    ]
  ])
})

function syntheticDeliveryStore(initialStatus?: 'claimed' | 'failed' | 'sent') {
  let now = Date.parse('2026-10-02T04:45:00Z')
  let status = initialStatus
  let claimedAt = now - 3 * 60_000
  let locked = false
  let sends = 0
  let failures = 0
  let previewsDisabled: boolean | undefined
  const bot = new Bot('123456:synthetic-test-token')
  bot.api.sendMessage = async (_chatId, _text, options) => {
    previewsDisabled = options?.link_preview_options?.is_disabled
    sends++
    if (failures-- > 0) throw new Error('Synthetic Telegram outage')
    return { message_id: 1, date: 1, chat: { id: 101, type: 'private', first_name: 'Test' }, text: 'lesson' }
  }
  const dependencies = {
    now: () => now,
    lease: async (_id: number, task: () => Promise<void>) => {
      if (locked) return false
      locked = true
      try {
        await task()
        return true
      } finally {
        locked = false
      }
    },
    loadUsers: async () => [{ telegramId: 101, groupName: 'QA' }],
    fetchSchedule: async () => ({
      group: 'QA',
      days: [
        {
          day: 'Friday',
          items: [
            {
              ...lesson,
              startsAt: '2026-10-02T05:00:00Z'
            }
          ]
        }
      ]
    }),
    claim: async (key: string, _telegramId: number, date: Date) => {
      if (status === 'sent' || (status === 'claimed' && date.getTime() - claimedAt < 2 * 60_000)) return undefined
      status = 'claimed'
      claimedAt = date.getTime()
      return { key, claimedAt: date.toISOString() }
    },
    finish: async (_claim: { key: string; claimedAt: string }, result: 'sent' | 'failed') => {
      status = result
    }
  }
  return {
    bot,
    dependencies,
    get sends() {
      return sends
    },
    get previewsDisabled() {
      return previewsDisabled
    },
    setNow: (date: number) => {
      now = date
    },
    failNext: () => {
      failures = 1
    }
  }
}

test('lesson deliveries allow Telegram to show the room link preview', async () => {
  const fixture = syntheticDeliveryStore()
  await checkAndNotify(fixture.bot, fixture.dependencies)
  assert.equal(fixture.sends, 1)
  assert.notEqual(fixture.previewsDisabled, true)
})

test('failed lesson deliveries retry before the deadline and successful deliveries never repeat', async () => {
  const fixture = syntheticDeliveryStore()
  fixture.failNext()
  const noDatabase = spyOn(db.query.telegramUsers, 'findMany').mockRejectedValue(new Error('Use synthetic store'))
  const noErrors = spyOn(console, 'error').mockImplementation(() => {})
  try {
    await checkAndNotify(fixture.bot, fixture.dependencies)
    fixture.setNow(Date.parse('2026-10-02T04:47:00Z'))
    await checkAndNotify(fixture.bot, fixture.dependencies)
    await checkAndNotify(fixture.bot, fixture.dependencies)
    assert.equal(fixture.sends, 2)
  } finally {
    noDatabase.mockRestore()
    noErrors.mockRestore()
  }
})

test('abandoned lesson claims recover but sent records and the prelesson deadline prevent resend', async () => {
  const noDatabase = spyOn(db.query.telegramUsers, 'findMany').mockRejectedValue(new Error('Use synthetic store'))
  const noErrors = spyOn(console, 'error').mockImplementation(() => {})
  try {
    const abandoned = syntheticDeliveryStore('claimed')
    await checkAndNotify(abandoned.bot, abandoned.dependencies)
    assert.equal(abandoned.sends, 1)
    const sent = syntheticDeliveryStore('sent')
    await checkAndNotify(sent.bot, sent.dependencies)
    assert.equal(sent.sends, 0)
    const expired = syntheticDeliveryStore('failed')
    expired.setNow(Date.parse('2026-10-02T04:59:00Z'))
    await checkAndNotify(expired.bot, expired.dependencies)
    assert.equal(expired.sends, 0)
  } finally {
    noDatabase.mockRestore()
    noErrors.mockRestore()
  }
})

test('concurrent lesson checks acquire the singleton lease before loading subscriptions', async () => {
  const fixture = syntheticDeliveryStore()
  let release: (() => void) | undefined
  const blocked = new Promise<void>((resolve) => {
    release = resolve
  })
  let loads = 0
  fixture.dependencies.loadUsers = async () => {
    loads++
    await blocked
    return [{ telegramId: 101, groupName: 'QA' }]
  }
  const noDatabase = spyOn(db.query.telegramUsers, 'findMany').mockRejectedValue(new Error('Use synthetic store'))
  const noErrors = spyOn(console, 'error').mockImplementation(() => {})
  try {
    const first = checkAndNotify(fixture.bot, fixture.dependencies)
    await checkAndNotify(fixture.bot, fixture.dependencies)
    release?.()
    await first
    assert.equal(loads, 1)
    assert.equal(fixture.sends, 1)
  } finally {
    release?.()
    noDatabase.mockRestore()
    noErrors.mockRestore()
  }
})

test('lesson delivery identity stays stable when a retry crosses UTC midnight', async () => {
  const fixture = syntheticDeliveryStore()
  fixture.setNow(Date.parse('2026-10-01T23:50:00Z'))
  fixture.dependencies.fetchSchedule = async () => ({
    group: 'QA',
    days: [
      {
        day: 'Friday',
        items: [
          {
            ...lesson,
            startsAt: '2026-10-02T00:05:00Z'
          }
        ]
      }
    ]
  })
  const keys: string[] = []
  const claim = fixture.dependencies.claim
  fixture.dependencies.claim = async (key, telegramId, now) => {
    keys.push(key)
    return claim(key, telegramId, now)
  }
  await checkAndNotify(fixture.bot, fixture.dependencies)
  fixture.setNow(Date.parse('2026-10-02T00:01:00Z'))
  await checkAndNotify(fixture.bot, fixture.dependencies)
  assert.equal(keys.length, 2)
  assert.equal(keys[0], keys[1])
  assert.equal(fixture.sends, 1)
})

test('SQL reclaim condition allows failed or old claimed rows while excluding sent rows', () => {
  const query = buildLessonClaim('synthetic-key', 101, new Date('2026-10-02T04:45:00Z')).toSQL()
  assert.match(query.sql, /on conflict.*do update.*where/)
  assert.match(query.sql, /"status" = \$\d+ or \("notification_deliveries"\."status" = \$\d+ and/)
  assert.ok(query.params.includes('failed'))
  assert.ok(query.params.includes('claimed'))
  assert.ok(query.params.includes('2026-10-02T04:43:00.000Z'))
  assert.ok(!query.params.includes('sent'))
})

test('lesson delivery rechecks its deadline after a slow claim', async () => {
  const fixture = syntheticDeliveryStore()
  fixture.setNow(Date.parse('2026-10-02T04:58:00Z'))
  const claim = fixture.dependencies.claim
  fixture.dependencies.claim = async (key, telegramId, now) => {
    const result = await claim(key, telegramId, now)
    fixture.setNow(Date.parse('2026-10-02T04:59:00Z'))
    return result
  }
  await checkAndNotify(fixture.bot, fixture.dependencies)
  assert.equal(fixture.sends, 0)
})
