import { spyOn } from 'bun:test'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Bot } from 'grammy'
import type { Chat } from 'grammy/types'

process.env.TELEGRAM_BOT_TOKEN ||= 'test-token'
const { setupCommands } = await import('./index')
const { db } = await import('../db')
const { env } = await import('../lib/env')
const { createMiniAppButton, getMiniAppMenuButton } = await import('../lib/mini-app')

function commandBot(menuFailure = false) {
  const messages: Array<{ text: string; replyMarkup: unknown }> = []
  const menus: unknown[] = []
  const bot = new Bot('123456:synthetic-test-token', {
    botInfo: {
      id: 123456,
      is_bot: true,
      first_name: 'Test',
      username: 'test_bot',
      can_join_groups: true,
      can_read_all_group_messages: false,
      supports_inline_queries: false,
      can_connect_to_business: false,
      has_main_web_app: false,
      has_topics_enabled: false,
      allows_users_to_create_topics: false
    }
  })
  bot.api.config.use(async (previous, method, payload) => {
    if (method === 'setChatMenuButton') {
      menus.push(JSON.parse(JSON.stringify(payload)))
      if (menuFailure) throw new Error('Synthetic menu update failure')
      return { ok: true, result: true } as Awaited<ReturnType<typeof previous<typeof method>>>
    }
    assert.equal(method, 'sendMessage', 'tests must only send intercepted synthetic replies')
    const message: { text: string; reply_markup?: unknown } = JSON.parse(JSON.stringify(payload))
    messages.push({ text: message.text, replyMarkup: message.reply_markup })
    // The runtime assertion narrows the synthetic result to sendMessage; grammY's generic method type cannot narrow.
    return {
      ok: true,
      result: {
        message_id: 1,
        date: 1,
        chat: { id: 101, type: 'private', first_name: 'Test' },
        text: message.text
      }
    } as Awaited<ReturnType<typeof previous<typeof method>>>
  })
  setupCommands(bot)
  return { bot, messages, menus }
}

async function sendCommand(bot: Bot, text: string, chatType: 'private' | 'group' = 'private') {
  const chat: Chat =
    chatType === 'private'
      ? { id: 101, type: 'private', first_name: 'Test' }
      : { id: -101, type: 'group', title: 'Test group' }
  await bot.handleUpdate({
    update_id: 1,
    message: {
      message_id: 1,
      date: 1,
      chat,
      from: { id: 101, is_bot: false, first_name: 'Test' },
      text,
      entities: [{ type: 'bot_command', offset: 0, length: text.length }]
    }
  })
}

test('/map opens the configured app without querying the database or schedule service', async () => {
  const userLookup = spyOn(db.query.telegramUsers, 'findFirst').mockImplementation(() => {
    throw new Error('map command must work independently of the database')
  })
  const fetchRequest = spyOn(globalThis, 'fetch').mockRejectedValue(new Error('No network calls allowed'))
  try {
    for (const chatType of ['private', 'group'] as const) {
      const { bot, messages } = commandBot()
      await sendCommand(bot, '/map', chatType)
      assert.equal(messages.length, 1)
      assert.deepEqual(messages[0].replyMarkup, {
        inline_keyboard: [[createMiniAppButton(env.MAP_APP_URL, chatType)]]
      })
    }
  } finally {
    userLookup.mockRestore()
    fetchRequest.mockRestore()
  }
})

test('/map replaces the private chat menu override and leaves group menus alone', async () => {
  for (const chatType of ['private', 'group'] as const) {
    const { bot, menus } = commandBot()
    await sendCommand(bot, '/map', chatType)
    assert.deepEqual(
      menus,
      chatType === 'private' ? [{ chat_id: 101, menu_button: getMiniAppMenuButton(env.MAP_APP_URL) }] : []
    )
  }
})

test('/map still offers the app when updating the menu fails', async () => {
  const warning = spyOn(console, 'warn').mockImplementation(() => {})
  try {
    const { bot, messages, menus } = commandBot(true)
    await sendCommand(bot, '/map')
    assert.equal(menus.length, 1)
    assert.equal(messages.length, 1)
    assert.deepEqual(messages[0].replyMarkup, {
      inline_keyboard: [[createMiniAppButton(env.MAP_APP_URL, 'private')]]
    })
  } finally {
    warning.mockRestore()
  }
})

test('/start exposes the app even when group selection cannot load', async () => {
  const insertUser = spyOn(db, 'insert').mockReturnValue({
    values: () => ({ onConflictDoNothing: async () => undefined })
  } as unknown as ReturnType<typeof db.insert>)
  const userLookup = spyOn(db.query.telegramUsers, 'findFirst').mockResolvedValue(undefined)
  const fetchRequest = spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Schedule unavailable'))
  const errorLog = spyOn(console, 'error').mockImplementation(() => {})
  try {
    const { bot, messages, menus } = commandBot()
    await sendCommand(bot, '/start')
    assert.deepEqual(menus, [{ chat_id: 101, menu_button: getMiniAppMenuButton(env.MAP_APP_URL) }])
    assert.deepEqual(messages[0].replyMarkup, {
      inline_keyboard: [[createMiniAppButton(env.MAP_APP_URL, 'private')]]
    })
    assert.match(messages[1].text, /Failed to load groups/)
  } finally {
    insertUser.mockRestore()
    userLookup.mockRestore()
    fetchRequest.mockRestore()
    errorLog.mockRestore()
  }
})

test('/help keeps slash command discovery available alongside the app', async () => {
  const { bot, messages } = commandBot()
  await sendCommand(bot, '/help')
  assert.match(messages[0].text, /\/map -/)
  assert.match(messages[0].text, /\/schedule -/)
})

test('/test opens its classroom directly from the notification', async () => {
  const { bot, messages } = commandBot()
  await sendCommand(bot, '/test')
  assert.deepEqual(messages[0].replyMarkup, {
    inline_keyboard: [[createMiniAppButton(env.MAP_APP_URL, 'private', { roomId: 'C1.2.301' })]]
  })
})

test('/schedule offers the map while the user has not selected a group', async () => {
  const userLookup = spyOn(db.query.telegramUsers, 'findFirst').mockResolvedValue(undefined)
  try {
    const { bot, messages } = commandBot()
    await sendCommand(bot, '/schedule')
    assert.deepEqual(messages[0].replyMarkup, {
      inline_keyboard: [[createMiniAppButton(env.MAP_APP_URL, 'private')]]
    })
  } finally {
    userLookup.mockRestore()
  }
})

test('/events adds contextual launches with slugs or numeric IDs and retains pagination', async () => {
  const fetchRequest = spyOn(globalThis, 'fetch').mockImplementation(
    Object.assign(
      async (request: Parameters<typeof fetch>[0]) => {
        assert.match(String(request instanceof Request ? request.url : request), /events.v1.EventsService\/ListEvents$/)
        return Response.json({
          events: [
            {
              id: 42,
              slug: 'open day/2026',
              title: 'Open day',
              description: '',
              startTime: new Date(Date.now() + 60_000).toISOString(),
              endTime: new Date(Date.now() + 120_000).toISOString(),
              status: 'EVENT_STATUS_PUBLISHED',
              organization: { title: 'Test' }
            },
            {
              id: 43,
              title: 'Another event',
              description: '',
              startTime: new Date(Date.now() + 120_000).toISOString(),
              endTime: new Date(Date.now() + 180_000).toISOString(),
              status: 'EVENT_STATUS_PUBLISHED',
              organization: { title: 'Test' }
            },
            {
              id: 44,
              title: 'Third event',
              status: 'EVENT_STATUS_PUBLISHED',
              startTime: new Date(Date.now() + 180_000).toISOString()
            },
            {
              id: 45,
              title: 'Fourth event',
              status: 'EVENT_STATUS_PUBLISHED',
              startTime: new Date(Date.now() + 240_000).toISOString()
            }
          ],
          total: 4
        })
      },
      { preconnect: globalThis.fetch.preconnect }
    )
  )
  try {
    const { bot, messages } = commandBot()
    await sendCommand(bot, '/events')
    assert.deepEqual(messages[0].replyMarkup, {
      inline_keyboard: [
        [createMiniAppButton(env.MAP_APP_URL, 'private', { eventSlug: 'open day/2026' }, '📌 Open day')],
        [createMiniAppButton(env.MAP_APP_URL, 'private', { eventSlug: '43' }, '📌 Another event')],
        [createMiniAppButton(env.MAP_APP_URL, 'private', { eventSlug: '44' }, '📌 Third event')],
        [
          { text: '1/2', callback_data: 'noop' },
          { text: 'Next ➡️', callback_data: 'events:page:1' }
        ]
      ]
    })
  } finally {
    fetchRequest.mockRestore()
  }
})
