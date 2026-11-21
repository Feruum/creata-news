import { bot } from './bot'
import { setupCallbacks } from './callbacks'
import { setupCommands } from './commands'
import { startEventReminderScheduler } from './features/reminders'
import { env } from './lib/env'
import { getMiniAppMenuButton } from './lib/mini-app'
import { markWebhookReady, startServer } from './server'
import { startScheduler } from './services/scheduler'

async function configureTelegramWebhook(): Promise<void> {
  if (!env.TELEGRAM_WEBHOOK_URL) {
    if (env.ENVIRONMENT !== 'development') {
      throw new Error('TELEGRAM_WEBHOOK_URL is required outside development')
    }
    console.warn('Telegram webhook URL is not configured; local HTTP endpoints remain available')
    markWebhookReady()
    return
  }

  await bot.api.setWebhook(env.TELEGRAM_WEBHOOK_URL, {
    secret_token: env.TELEGRAM_WEBHOOK_SECRET,
    drop_pending_updates: false,
    max_connections: 100
  })
  markWebhookReady()
  console.log(`Telegram webhook configured for ${env.TELEGRAM_WEBHOOK_URL}`)
}

async function main() {
  setupCommands(bot)
  setupCallbacks(bot)
  startServer()

  await bot.init()
  await bot.api.setMyCommands([
    { command: 'start', description: 'Start the bot and select your group' },
    { command: 'map', description: 'Открыть карту кампуса, расписание и события' },
    { command: 'settings', description: 'Change your group or notification settings' },
    { command: 'schedule', description: "View today's schedule" },
    { command: 'events', description: 'View upcoming campus events' },
    { command: 'reminders', description: 'Manage event reminder preferences' },
    { command: 'test', description: 'Send a test notification' },
    { command: 'help', description: 'Show help message' }
  ])
  await bot.api.setChatMenuButton({ menu_button: getMiniAppMenuButton(env.MAP_APP_URL) })

  await configureTelegramWebhook()
  startScheduler(bot)
  startEventReminderScheduler(bot)
}

main().catch((error) => {
  console.error('Telegram bot startup failed', error)
  process.exitCode = 1
})
