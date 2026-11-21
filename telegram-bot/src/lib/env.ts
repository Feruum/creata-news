import { z } from 'zod'
import { DEFAULT_MAP_APP_URL, mapAppUrlSchema } from './mini-app'

const envSchema = z
  .object({
    TELEGRAM_BOT_TOKEN: z.string().min(1, 'TELEGRAM_BOT_TOKEN is required'),
    INTERNAL_REMINDER_SECRET: z.string().default(''),
    TELEGRAM_WEBHOOK_URL: z.string().url().optional(),
    TELEGRAM_WEBHOOK_SECRET: z.string().min(16).default('local-webhook-secret'),
    ENVIRONMENT: z.enum(['development', 'staging', 'production']).default('development'),
    DATABASE_URL: z.string().default('postgres://postgres:postgres@localhost:5432/telegram_bot'),
    SCHEDULE_API_URL: z.string().default('http://localhost:8081'),
    EMS_API_URL: z.string().default('https://api.studyverse.app'),
    MAP_APP_URL: mapAppUrlSchema.default(DEFAULT_MAP_APP_URL),
    TIMEZONE: z.string().default('Asia/Almaty'),
    HTTP_PORT: z.coerce.number().default(3001)
  })
  .superRefine((value, context) => {
    if (value.ENVIRONMENT !== 'development' && value.INTERNAL_REMINDER_SECRET.length < 32) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['INTERNAL_REMINDER_SECRET'],
        message: 'INTERNAL_REMINDER_SECRET must contain at least 32 characters outside development'
      })
    }
    if (value.ENVIRONMENT !== 'development' && !value.TELEGRAM_WEBHOOK_URL) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['TELEGRAM_WEBHOOK_URL'],
        message: 'TELEGRAM_WEBHOOK_URL is required outside development'
      })
    }
    if (value.ENVIRONMENT !== 'development' && value.TELEGRAM_WEBHOOK_SECRET.length < 32) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['TELEGRAM_WEBHOOK_SECRET'],
        message: 'TELEGRAM_WEBHOOK_SECRET must contain at least 32 characters outside development'
      })
    }
  })

export const env = envSchema.parse(process.env)
