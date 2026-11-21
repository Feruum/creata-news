import { AbortController } from 'abort-controller'
import { Cron } from 'croner'
import { and, eq, lt, or } from 'drizzle-orm'
import type { Bot, Context } from 'grammy'
import { InlineKeyboard } from 'grammy'
import { db, schema } from '../db'
import { env } from '../lib/env'
import { createMiniAppButton, generateMiniAppUrl } from '../lib/mini-app'
import { isLessonReminderDue } from './lesson-time'
import { fetchGroupSchedule, type ScheduleItem } from './schedule'
import { withSingletonLease } from './singleton'

function getNotificationKey(telegramId: number, lesson: ScheduleItem): string {
  return `${telegramId}:${lesson.startsAt}:${lesson.subject}:${lesson.room}`
}

export function createLessonNotification(lesson: ScheduleItem, baseUrl = env.MAP_APP_URL, now = Date.now()) {
  const target = { roomId: lesson.mapRoomId || '' }
  const mapLink = generateMiniAppUrl(baseUrl, target)

  const minutes = lesson.startsAt ? Math.ceil((Date.parse(lesson.startsAt) - now) / 60_000) : 15
  const text =
    `📚 Пара через ${minutes} минут!\n\n` +
    `📖 ${lesson.subject}\n` +
    `👨‍🏫 ${lesson.tutor}\n` +
    `🏛 ${lesson.room}\n` +
    `📝 ${lesson.lessonType}\n` +
    `⏰ ${lesson.classtimeTime} (UTC+5)\n\n` +
    `🗺 ${mapLink}`
  return {
    text,
    reply_markup: new InlineKeyboard().add(createMiniAppButton(baseUrl, 'private', target))
  }
}

interface DeliveryClaim {
  key: string
  claimedAt: string
}

interface LessonNotificationDependencies {
  now: () => number
  lease: typeof withSingletonLease
  loadUsers: () => Promise<Array<{ telegramId: number; groupName: string | null }>>
  fetchSchedule: typeof fetchGroupSchedule
  claim: (key: string, telegramId: number, now: Date) => Promise<DeliveryClaim | undefined>
  finish: (claim: DeliveryClaim, status: 'sent' | 'failed', error?: string) => Promise<void>
}

export function buildLessonClaim(key: string, telegramId: number, now: Date) {
  const deliveries = schema.notificationDeliveries
  return db
    .insert(deliveries)
    .values({
      deduplicationKey: key,
      notificationType: 'lesson_reminder',
      telegramId,
      claimedAt: now.toISOString()
    })
    .onConflictDoUpdate({
      target: deliveries.deduplicationKey,
      set: { status: 'claimed', claimedAt: now.toISOString(), errorMessage: null },
      setWhere: or(
        eq(deliveries.status, 'failed'),
        and(
          eq(deliveries.status, 'claimed'),
          lt(deliveries.claimedAt, new Date(now.getTime() - 2 * 60_000).toISOString())
        )
      )
    })
    .returning({ key: deliveries.deduplicationKey, claimedAt: deliveries.claimedAt })
}

const defaultDependencies: LessonNotificationDependencies = {
  now: Date.now,
  lease: withSingletonLease,
  loadUsers: () => db.query.telegramUsers.findMany({ where: eq(schema.telegramUsers.notificationsEnabled, true) }),
  fetchSchedule: fetchGroupSchedule,
  claim: async (key, telegramId, now) => (await buildLessonClaim(key, telegramId, now))[0],
  finish: async (claim, status, error) => {
    await db
      .update(schema.notificationDeliveries)
      .set({
        status,
        sentAt: status === 'sent' ? new Date().toISOString() : null,
        errorMessage: error || null
      })
      .where(
        and(
          eq(schema.notificationDeliveries.deduplicationKey, claim.key),
          eq(schema.notificationDeliveries.status, 'claimed'),
          eq(schema.notificationDeliveries.claimedAt, claim.claimedAt)
        )
      )
  }
}

export async function checkAndNotify(bot: Bot<Context>, overrides: Partial<LessonNotificationDependencies> = {}) {
  const dependencies = { ...defaultDependencies, ...overrides }
  try {
    await dependencies.lease(7_302_001, () => processLessonNotifications(bot, dependencies))
  } catch (error) {
    console.error('Scheduler error:', error)
  }
}

async function processLessonNotifications(bot: Bot<Context>, dependencies: LessonNotificationDependencies) {
  try {
    const users = await dependencies.loadUsers()

    const groupSchedules = new Map<string, ScheduleItem[]>()

    for (const user of users) {
      if (!user.groupName) continue

      let todayLessons = groupSchedules.get(user.groupName)

      if (!todayLessons) {
        try {
          const schedule = await dependencies.fetchSchedule(user.groupName)
          todayLessons = schedule.days.flatMap((day) => day.items)
          groupSchedules.set(user.groupName, todayLessons)
        } catch (error) {
          console.error(`Failed to fetch schedule for ${user.groupName}:`, error)
          continue
        }
      }

      for (const lesson of todayLessons) {
        const now = dependencies.now()
        if (!lesson.startsAt || !isLessonReminderDue(lesson.startsAt, now)) continue

        const key = getNotificationKey(user.telegramId, lesson)
        // Preserve the original due-date key across midnight and across retries.
        const dueDate = new Date(Date.parse(lesson.startsAt) - 15 * 60_000).toISOString().slice(0, 10)
        const claim = await dependencies.claim(`${dueDate}:${key}`, user.telegramId, new Date(now))
        if (!claim) continue

        const sendNow = dependencies.now()
        if (!isLessonReminderDue(lesson.startsAt, sendNow)) {
          await dependencies.finish(claim, 'failed', 'Prelesson delivery deadline passed')
          continue
        }

        const controller = new AbortController()
        const timer = setTimeout(
          () => controller.abort(),
          Math.min(30_000, Date.parse(lesson.startsAt) - sendNow - 60_000)
        )
        try {
          const { text, reply_markup } = createLessonNotification(lesson, env.MAP_APP_URL, sendNow)
          await bot.api.sendMessage(
            user.telegramId,
            text,
            {
              reply_markup,
              link_preview_options: { is_disabled: false }
            },
            controller.signal
          )
        } catch (error) {
          await dependencies.finish(claim, 'failed', error instanceof Error ? error.message : 'Unknown error')
          console.error(`Failed to send notification to ${user.telegramId}:`, error)
          continue
        } finally {
          clearTimeout(timer)
        }
        // Persistence errors after a successful Telegram send must never turn the delivery into a failed send.
        await dependencies.finish(claim, 'sent')
      }
    }
  } catch (error) {
    console.error('Scheduler error:', error)
  }
}

export function startScheduler(bot: Bot<Context>) {
  new Cron('* * * * *', { timezone: env.TIMEZONE, protect: true }, async () => {
    await checkAndNotify(bot)
  })

  console.log(`Scheduler started (timezone: ${env.TIMEZONE})`)
}
