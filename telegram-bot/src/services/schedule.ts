import { z } from 'zod'
import { env } from '../lib/env'
import { campusDate } from './lesson-time'

export interface ScheduleItem {
  startsAt?: string
  mapRoomId?: string | null
  subject: string
  tutor: string
  classtimeDay: string
  classtimeTime: string
  lessonType: string
  room: string
  group: string
  teamsMeetingUrl: string
}

export interface ScheduleDay {
  date?: string
  day: string
  items: ScheduleItem[]
}

export interface GroupScheduleResponse {
  group: string
  days: ScheduleDay[]
}

export async function fetchGroups(): Promise<string[]> {
  const response = await fetch(`${env.MAP_APP_URL}/api/schedule/groups`, { signal: AbortSignal.timeout(10_000) })
  if (!response.ok) {
    throw new Error(`Failed to fetch groups: ${response.statusText}`)
  }
  return z.array(z.string().max(40)).parse(await response.json())
}

export async function fetchGroupSchedule(groupName: string): Promise<GroupScheduleResponse> {
  const response = await fetch(`${env.MAP_APP_URL}/api/schedule/groups?group=${encodeURIComponent(groupName)}`, {
    signal: AbortSignal.timeout(10_000)
  })
  if (!response.ok) {
    throw new Error(`Failed to fetch schedule for ${groupName}: ${response.statusText}`)
  }
  const snapshot = z
    .object({
      groupName: z.string(),
      events: z.array(
        z.object({
          title: z.string(),
          teacher: z.string(),
          start: z.string().datetime({ offset: true }),
          end: z.string().datetime({ offset: true }),
          lessonType: z.string(),
          location: z.string(),
          mapRoomId: z.string().nullable().optional()
        })
      )
    })
    .parse(await response.json())
  const days = new Map<string, ScheduleDay>()
  const clock = (value: string) =>
    new Intl.DateTimeFormat('en-GB', { timeZone: 'Asia/Almaty', hour: '2-digit', minute: '2-digit' }).format(
      new Date(value)
    )
  for (const event of snapshot.events) {
    const date = campusDate(new Date(event.start))
    const day = days.get(date) || {
      date,
      day: new Intl.DateTimeFormat('en-US', { timeZone: 'Asia/Almaty', weekday: 'long' }).format(new Date(event.start)),
      items: []
    }
    day.items.push({
      subject: event.title,
      tutor: event.teacher,
      startsAt: event.start,
      classtimeDay: day.day,
      classtimeTime: `${clock(event.start)}–${clock(event.end)}`,
      lessonType: event.lessonType,
      room: event.location,
      mapRoomId: event.mapRoomId,
      group: snapshot.groupName,
      teamsMeetingUrl: ''
    })
    days.set(date, day)
  }
  return { group: snapshot.groupName, days: [...days.values()] }
}

export function parseTimeRange(classtimeTime: string): { start: string; end: string } | null {
  const match = classtimeTime.match(/(\d{2}:\d{2})\s*[-–]\s*(\d{2}:\d{2})/)
  if (!match) return null
  return { start: match[1], end: match[2] }
}

export function getCurrentDayName(): string {
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
  return days[new Date().getDay()]
}

export function getTodayLessons(schedule: GroupScheduleResponse): ScheduleItem[] {
  const today = getCurrentDayName()
  const todaySchedule = schedule.days.find((d) =>
    d.date ? d.date === campusDate() : d.day.toLowerCase() === today.toLowerCase()
  )
  return todaySchedule?.items || []
}
