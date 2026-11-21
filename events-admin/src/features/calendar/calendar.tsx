'use client'

import { createClient } from '@connectrpc/connect'
import { createConnectQueryKey, useTransport } from '@connectrpc/connect-query'
import type { Event } from '@studyverse/proto'
import { EventsService, listEvents, listEventsForAdmin } from '@studyverse/proto'
import { useSuspenseQuery } from '@tanstack/react-query'
import { differenceInHours, parseISO } from 'date-fns'
import { Suspense } from 'react'
import { CalendarBody } from '@/features/calendar/calendar-body'
import { CalendarProvider } from '@/features/calendar/contexts/calendar-context'
import { DndProvider } from '@/features/calendar/contexts/dnd-context'
import { CalendarHeader } from '@/features/calendar/header/calendar-header'
import { hexToEventColor } from '@/features/calendar/helpers'
import type { ICategory, IEvent, IUser } from '@/features/calendar/interfaces'
import { CalendarSkeleton } from '@/features/calendar/skeletons/calendar-skeleton'
import { getEventDescriptionText } from '@/features/events'
import { useAuth } from '@/shared/lib/auth'
import { loadCalendarEvents } from './api/load-calendar-events'
import { CalendarPersistenceProvider } from './calendar-persistence-provider'

function transformEventToCalendarEvent(event: Event): IEvent {
  const start = parseISO(event.startTime)
  const end = parseISO(event.endTime)
  const isAllDay = differenceInHours(end, start) >= 24

  const category: ICategory | null = event.category
    ? {
        id: event.category.id,
        name: event.category.name,
        color: event.category.color
      }
    : null

  const hasCategory = category?.color

  return {
    id: Number(event.id),
    title: event.title,
    description: getEventDescriptionText(event.description),
    startDate: event.startTime,
    endDate: event.endTime,
    color: hasCategory ? hexToEventColor(category.color) : 'gray',
    hexColor: hasCategory ? category.color : null,
    isAllDay,
    user: {
      id: String(event.userId),
      name: event.organization?.title || 'Unknown',
      picturePath: event.organization?.imageUrl || null
    },
    category
  }
}

function CalendarContent() {
  const transport = useTransport()
  const { data } = useSuspenseQuery({
    queryKey: [...createConnectQueryKey({ schema: listEventsForAdmin, cardinality: 'finite' }), 'calendar-all-pages'],
    queryFn: ({ signal }) =>
      loadCalendarEvents((input) => createClient(EventsService, transport).listEventsForAdmin(input, { signal }))
  })

  return <CalendarEvents events={data.events} />
}

function GuestCalendarContent() {
  const transport = useTransport()
  const { data } = useSuspenseQuery({
    queryKey: [...createConnectQueryKey({ schema: listEvents, cardinality: 'finite' }), 'calendar-all-pages'],
    queryFn: ({ signal }) =>
      loadCalendarEvents((input) => createClient(EventsService, transport).listEvents(input, { signal }))
  })

  return <CalendarEvents events={data.events} />
}

function CalendarEvents({ events: sourceEvents }: { events: Event[] }) {
  const events: IEvent[] = sourceEvents.map(transformEventToCalendarEvent)

  const usersMap = new Map<string, IUser>()
  events.forEach((event) => {
    if (!usersMap.has(event.user.id)) {
      usersMap.set(event.user.id, event.user)
    }
  })
  const users: IUser[] = Array.from(usersMap.values())

  return (
    <CalendarProvider events={events} users={users} view='month'>
      <CalendarPersistenceProvider>
        <DndProvider showConfirmation={false}>
          <div className='w-full border rounded-xl'>
            <CalendarHeader />
            <CalendarBody />
          </div>
        </DndProvider>
      </CalendarPersistenceProvider>
    </CalendarProvider>
  )
}

export function Calendar() {
  const { isAuthenticated, isOfflineReadonly } = useAuth()

  return (
    <Suspense fallback={<CalendarSkeleton />}>
      {isAuthenticated || isOfflineReadonly ? <CalendarContent /> : <GuestCalendarContent />}
    </Suspense>
  )
}
