'use client'

import { useContext } from 'react'
import type { ICalendarContext } from './calendar-context'
import { CalendarContext } from './calendar-context'

export function useCalendar(): ICalendarContext {
  const context = useContext(CalendarContext)
  if (!context) throw new Error('useCalendar must be used within a CalendarProvider.')
  return context
}
