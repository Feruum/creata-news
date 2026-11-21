import { CheckInPhase, type Event, EventFormat, EventStatus as ProtoEventStatus } from '@studyverse/proto'

import type { EventStatus } from '@/shared/ui/components/event-status-badge'

export const statusOptions = [
  { value: 'draft', label: 'Draft' },
  { value: 'live', label: 'Live' },
  { value: 'today', label: 'Today' },
  { value: 'upcoming', label: 'Upcoming' },
  { value: 'past', label: 'Past' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'archived', label: 'Archived' }
]

export function getEventStatus(event: Pick<Event, 'status' | 'checkInPhase' | 'startTime' | 'endTime'>): EventStatus {
  if (event.status === ProtoEventStatus.DRAFT) return 'draft'
  if (event.status === ProtoEventStatus.CANCELLED) return 'cancelled'
  if (event.status === ProtoEventStatus.ARCHIVED) return 'archived'

  if (event.checkInPhase === CheckInPhase.LIVE || event.checkInPhase === CheckInPhase.GRACE) {
    return 'live'
  }
  if (event.checkInPhase === CheckInPhase.LOCKED) {
    return 'past'
  }

  const now = new Date()
  const startTime = new Date(event.startTime)
  const endTime = event.endTime ? new Date(event.endTime) : startTime

  if (now >= startTime && now <= endTime) {
    return 'live'
  }

  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const todayEnd = new Date(todayStart)
  todayEnd.setDate(todayEnd.getDate() + 1)

  if (startTime >= todayStart && startTime < todayEnd && now < startTime) {
    return 'today'
  }

  if (endTime < now) {
    return 'past'
  }

  return 'upcoming'
}

export const formatOptions = [
  { value: EventFormat.ONLINE, label: 'Online' },
  { value: EventFormat.OFFLINE, label: 'Offline' }
]
