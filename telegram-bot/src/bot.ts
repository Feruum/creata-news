import { autoRetry } from '@grammyjs/auto-retry'
import { Bot } from 'grammy'
import { env } from './lib/env'

export const bot = new Bot(env.TELEGRAM_BOT_TOKEN)

bot.api.config.use(autoRetry())
