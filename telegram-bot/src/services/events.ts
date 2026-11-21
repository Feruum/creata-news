import { EventStatus, LocationType, type Event as ProtoEvent } from '@studyverse/proto'
import { plateJsonToText } from '../lib/plate-utils'
import { eventsClient } from '../lib/rpc'

export interface Event {
  id: string
  slug?: string
  title: string
  description: string
  startDateTime: string
  endDateTime: string
  location: string
  organization: string
  imageUrl?: string
}

export interface EventsResponse {
  events: Event[]
  total: number
}

export async function fetchUpcomingEvents(
  limit = 10,
  offset = 0,
  dependencies: { client: Pick<typeof eventsClient, 'listEvents'>; now: () => number } = {
    client: eventsClient,
    now: Date.now
  }
): Promise<EventsResponse> {
  if (!Number.isSafeInteger(limit) || limit < 1 || !Number.isSafeInteger(offset) || offset < 0) {
    throw new Error('Invalid event pagination')
  }
  const now = dependencies.now()
  const upcoming = new Map<number, ProtoEvent>()
  // ListEvents is ordered by ID, so time filtering and pagination must follow the complete scan.
  let scanned = 0
  for (let page = 1; ; page++) {
    const response = await dependencies.client.listEvents(
      { page, limit: 100, status: EventStatus.PUBLISHED },
      { timeoutMs: 10_000 }
    )
    for (const event of response.events) {
      if (event.status === EventStatus.PUBLISHED && Date.parse(event.startTime) > now) upcoming.set(event.id, event)
    }
    scanned += response.events.length
    if (response.events.length === 0 || scanned >= response.total) break
  }
  const events = [...upcoming.values()].sort((a, b) => Date.parse(a.startTime) - Date.parse(b.startTime) || a.id - b.id)
  return {
    total: events.length,
    events: events.slice(offset, offset + limit).map((event) => {
      const location = event.structuredLocation
      const locationText = !location
        ? ''
        : location.type === LocationType.ONLINE
          ? location.onlinePlatform || location.onlineUrl || 'Online'
          : location.type === LocationType.EXTERNAL
            ? location.externalAddress || ''
            : [
                location.roomName || location.roomId,
                location.type === LocationType.HYBRID && (location.onlinePlatform || location.onlineUrl)
              ]
                .filter(Boolean)
                .join(' / ')
      return {
        id: String(event.id),
        slug: event.slug,
        title: event.title,
        description: event.description,
        startDateTime: event.startTime,
        endDateTime: event.endTime,
        imageUrl: event.imageUrl,
        location: locationText,
        organization: event.organization?.title || ''
      }
    })
  }
}

export function formatEventDateTime(dateTimeStr: string): string {
  try {
    const date = new Date(dateTimeStr)
    const time = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })
    const dayDate = date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    return `${dayDate} ${time}`
  } catch {
    return dateTimeStr
  }
}

export function formatEventMessage(event: Event): string {
  const dateTime = formatEventDateTime(event.startDateTime)
  let message = `📌 *${event.title}*\n`
  message += `⏰ ${dateTime}\n`
  message += `📍 ${event.location}\n`
  message += `🏢 ${event.organization}`
  const descriptionText = plateJsonToText(event.description)
  if (descriptionText) {
    message += `\n\n${descriptionText}`
  }
  return message
}
