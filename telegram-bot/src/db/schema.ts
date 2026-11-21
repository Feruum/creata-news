import { bigint, boolean, pgTable, serial, text, timestamp } from 'drizzle-orm/pg-core'

export const telegramUsers = pgTable('telegram_users', {
  id: serial('id').primaryKey(),
  telegramId: bigint('telegram_id', { mode: 'number' }).notNull().unique(),
  username: text('username'),
  firstName: text('first_name'),
  lastName: text('last_name'),
  groupName: text('group_name'),
  notificationsEnabled: boolean('notifications_enabled').default(true).notNull(),
  reminderDayBefore: boolean('reminder_day_before').default(true).notNull(),
  reminderHourBefore: boolean('reminder_hour_before').default(true).notNull(),
  reminder15Min: boolean('reminder_15_min').default(false).notNull(),
  createdAt: timestamp('created_at', { withTimezone: true, mode: 'string' }).defaultNow().notNull(),
  updatedAt: timestamp('updated_at', { withTimezone: true, mode: 'string' }).defaultNow().notNull()
})

export const notificationDeliveries = pgTable('notification_deliveries', {
  deduplicationKey: text('deduplication_key').primaryKey(),
  notificationType: text('notification_type').notNull(),
  telegramId: bigint('telegram_id', { mode: 'number' }).notNull(),
  status: text('status').default('claimed').notNull(),
  claimedAt: timestamp('claimed_at', { withTimezone: true, mode: 'string' }).defaultNow().notNull(),
  sentAt: timestamp('sent_at', { withTimezone: true, mode: 'string' }),
  errorMessage: text('error_message')
})

export type TelegramUser = typeof telegramUsers.$inferSelect
export type NewTelegramUser = typeof telegramUsers.$inferInsert
export type NotificationDelivery = typeof notificationDeliveries.$inferSelect
