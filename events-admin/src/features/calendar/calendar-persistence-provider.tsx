import { useRef, type ReactNode } from 'react'
import { useUpdateEvent } from '@/features/events'
import { CalendarContext } from './contexts/calendar-context'
import { useCalendar } from './contexts/use-calendar'
import { persistCalendarEvent } from './event-persistence'
import type { IEvent } from './interfaces'

export function CalendarPersistenceProvider({ children }: { children: ReactNode }) {
  const calendar = useCalendar()
  const mutation = useUpdateEvent()
  const pending = useRef(new Set<number>())

  const updateEvent = async (event: IEvent) => {
    const original = calendar.events.find((item) => item.id === event.id)
    if (!original) throw new Error('Event is no longer available')
    if (pending.current.has(event.id)) throw new Error('Please wait for the previous change to save')
    pending.current.add(event.id)
    try {
      await persistCalendarEvent(original, event, calendar.updateEvent, mutation.mutateAsync)
    } finally {
      pending.current.delete(event.id)
    }
  }

  return <CalendarContext.Provider value={{ ...calendar, updateEvent }}>{children}</CalendarContext.Provider>
}
