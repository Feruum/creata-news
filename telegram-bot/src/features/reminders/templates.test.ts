import assert from 'node:assert/strict'
import { test } from 'node:test'
import { LocationType } from '@studyverse/proto'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { getReminderKeyboard, formatReminderMessage } = await import('./templates')

test('reminder content cannot break Telegram HTML formatting', () => {
  const message = formatReminderMessage('15_min', {
    eventTitle: 'Workshop <intro> & Q&A',
    organizationTitle: 'Club <A>',
    startTime: '2026-10-02T21:00:00Z',
    endTime: '2026-10-02T22:00:00Z',
    slug: 'workshop',
    location: {
      $typeName: 'events.v1.EventLocation',
      type: LocationType.ONLINE,
      onlineUrl: 'https://example.test/join?a=1&b=2',
      onlinePlatform: 'Meet <live>'
    }
  })
  assert.match(message, /Workshop &lt;intro&gt; &amp; Q&amp;A/)
  assert.match(message, /Club &lt;A&gt;/)
  assert.match(message, /href="https:\/\/example.test\/join\?a=1&amp;b=2"/)
  assert.match(message, /Meet &lt;live&gt;/)
})

test('reminder cancellation opens the authenticated Mini App event screen with explicit instructions', () => {
  const keyboard = getReminderKeyboard(42, 'open day/2026', 'private', 'https://map.webspace.cat')
  assert.deepEqual(keyboard.inline_keyboard, [
    [{ text: '✅ Confirm', callback_data: 'reminder:confirm:42' }],
    [
      {
        text: '❌ Open event to cancel registration',
        web_app: { url: 'https://map.webspace.cat/event/open%20day%2F2026' }
      }
    ]
  ])
})

test('group reminders keep a normal link and missing slugs keep only the callbacks', () => {
  assert.deepEqual(getReminderKeyboard(42, 'event', 'group', 'https://map.webspace.cat').inline_keyboard[1], [
    { text: '❌ Open event to cancel registration', url: 'https://map.webspace.cat/event/event' }
  ])
  assert.equal(getReminderKeyboard(42, '', 'private', 'https://map.webspace.cat').inline_keyboard.length, 1)
})

test('reminder text explains the authenticated cancellation step', () => {
  const text = formatReminderMessage('hour_before', {
    eventTitle: 'Event',
    organizationTitle: 'Club',
    slug: 'event',
    startTime: '2026-10-02T05:00:00Z',
    endTime: '2026-10-02T06:00:00Z'
  })
  assert.match(text, /open the event, sign in, and choose Cancel registration/i)
})
