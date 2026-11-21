'use client'

import type React from 'react'
import { createContext, useEffect, useMemo, useRef, useState } from 'react'
import { useLocalStorage } from '@/features/calendar/hooks'
import type { ICategory, IEvent, IUser } from '@/features/calendar/interfaces'
import type { TCalendarView, TEventColor } from '@/features/calendar/types'

export interface ICalendarContext {
  selectedDate: Date
  view: TCalendarView
  setView: (view: TCalendarView) => void
  agendaModeGroupBy: 'date' | 'category'
  setAgendaModeGroupBy: (groupBy: 'date' | 'category') => void
  use24HourFormat: boolean
  toggleTimeFormat: () => void
  setSelectedDate: (date: Date | undefined) => void
  selectedUserId: IUser['id'] | 'all'
  setSelectedUserId: (userId: IUser['id'] | 'all') => void
  badgeVariant: 'dot' | 'colored'
  setBadgeVariant: (variant: 'dot' | 'colored') => void
  selectedColors: TEventColor[]
  filterEventsBySelectedColors: (colors: TEventColor) => void
  filterEventsBySelectedUser: (userId: IUser['id'] | 'all') => void
  users: IUser[]
  events: IEvent[]
  addEvent: (event: IEvent) => void
  updateEvent: (event: IEvent) => void | Promise<void>
  removeEvent: (eventId: number) => void
  clearFilter: () => void
  categories: ICategory[]
  selectedCategoryIds: number[]
  filterEventsByCategory: (categoryId: number) => void
}

interface CalendarSettings {
  badgeVariant: 'dot' | 'colored'
  view: TCalendarView
  use24HourFormat: boolean
  agendaModeGroupBy: 'date' | 'category'
}

const DEFAULT_SETTINGS: CalendarSettings = {
  badgeVariant: 'colored',
  view: 'day',
  use24HourFormat: true,
  agendaModeGroupBy: 'date'
}

// eslint-disable-next-line react-refresh/only-export-components
export const CalendarContext = createContext({} as ICalendarContext)

export function CalendarProvider({
  children,
  users,
  events,
  badge = 'colored',
  view = 'day'
}: {
  children: React.ReactNode
  users: IUser[]
  events: IEvent[]
  view?: TCalendarView
  badge?: 'dot' | 'colored'
}) {
  const [settings, setSettings] = useLocalStorage<CalendarSettings>('calendar-settings', {
    ...DEFAULT_SETTINGS,
    badgeVariant: badge,
    view: view
  })

  const [badgeVariant, setBadgeVariantState] = useState<'dot' | 'colored'>(settings.badgeVariant)
  const [currentView, setCurrentViewState] = useState<TCalendarView>(settings.view)
  const [use24HourFormat, setUse24HourFormatState] = useState<boolean>(settings.use24HourFormat)
  const [agendaModeGroupBy, setAgendaModeGroupByState] = useState<'date' | 'category'>(settings.agendaModeGroupBy)

  const [selectedDate, setSelectedDate] = useState(new Date())
  const [selectedUserId, setSelectedUserId] = useState<IUser['id'] | 'all'>('all')
  const [selectedColors, setSelectedColors] = useState<TEventColor[]>([])
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<number[]>([])

  const [allEvents, setAllEvents] = useState<IEvent[]>(events || [])
  const [filteredEvents, setFilteredEvents] = useState<IEvent[]>(events || [])
  const activeFilters = useRef({ selectedColors, selectedUserId, selectedCategoryIds })
  activeFilters.current = { selectedColors, selectedUserId, selectedCategoryIds }

  useEffect(() => {
    const { selectedColors, selectedUserId, selectedCategoryIds } = activeFilters.current
    setAllEvents(events)
    setFilteredEvents(
      events.filter(
        (event) =>
          (selectedColors.length === 0 || selectedColors.includes(event.color || 'blue')) &&
          (selectedUserId === 'all' || event.user.id === selectedUserId) &&
          (selectedCategoryIds.length === 0 || (event.category && selectedCategoryIds.includes(event.category.id)))
      )
    )
    // Refetches replace server data while preserving the current view and filters.
  }, [events])

  const categories = useMemo(() => {
    const categoryMap = new Map<number, ICategory>()
    allEvents.forEach((event) => {
      if (event.category) {
        categoryMap.set(event.category.id, event.category)
      }
    })
    return Array.from(categoryMap.values())
  }, [allEvents])

  const updateSettings = (newPartialSettings: Partial<CalendarSettings>) => {
    setSettings({
      ...settings,
      ...newPartialSettings
    })
  }

  const setBadgeVariant = (variant: 'dot' | 'colored') => {
    setBadgeVariantState(variant)
    updateSettings({ badgeVariant: variant })
  }

  const setView = (newView: TCalendarView) => {
    setCurrentViewState(newView)
    updateSettings({ view: newView })
  }

  const toggleTimeFormat = () => {
    const newValue = !use24HourFormat
    setUse24HourFormatState(newValue)
    updateSettings({ use24HourFormat: newValue })
  }

  const setAgendaModeGroupBy = (groupBy: 'date' | 'category') => {
    setAgendaModeGroupByState(groupBy)
    updateSettings({ agendaModeGroupBy: groupBy })
  }

  const filterEventsBySelectedColors = (color: TEventColor) => {
    const isColorSelected = selectedColors.includes(color)
    const newColors = isColorSelected ? selectedColors.filter((c) => c !== color) : [...selectedColors, color]

    if (newColors.length > 0) {
      const filtered = allEvents.filter((event) => {
        const eventColor = event.color || 'blue'
        return newColors.includes(eventColor)
      })
      setFilteredEvents(filtered)
    } else {
      setFilteredEvents(allEvents)
    }

    setSelectedColors(newColors)
  }

  const filterEventsBySelectedUser = (userId: IUser['id'] | 'all') => {
    setSelectedUserId(userId)
    if (userId === 'all') {
      setFilteredEvents(allEvents)
    } else {
      const filtered = allEvents.filter((event) => event.user.id === userId)
      setFilteredEvents(filtered)
    }
  }

  const handleSelectDate = (date: Date | undefined) => {
    if (!date) return
    setSelectedDate(date)
  }

  const addEvent = (event: IEvent) => {
    setAllEvents((prev) => [...prev, event])
    setFilteredEvents((prev) => [...prev, event])
  }

  const updateEvent = (event: IEvent) => {
    const updated = {
      ...event,
      startDate: new Date(event.startDate).toISOString(),
      endDate: new Date(event.endDate).toISOString()
    }

    setAllEvents((prev) => prev.map((e) => (e.id === event.id ? updated : e)))
    setFilteredEvents((prev) => prev.map((e) => (e.id === event.id ? updated : e)))
  }

  const removeEvent = (eventId: number) => {
    setAllEvents((prev) => prev.filter((e) => e.id !== eventId))
    setFilteredEvents((prev) => prev.filter((e) => e.id !== eventId))
  }

  const clearFilter = () => {
    setFilteredEvents(allEvents)
    setSelectedColors([])
    setSelectedCategoryIds([])
    setSelectedUserId('all')
  }

  const filterEventsByCategory = (categoryId: number) => {
    const isSelected = selectedCategoryIds.includes(categoryId)
    const newCategoryIds = isSelected
      ? selectedCategoryIds.filter((id) => id !== categoryId)
      : [...selectedCategoryIds, categoryId]

    if (newCategoryIds.length > 0) {
      const filtered = allEvents.filter((event) => {
        return event.category && newCategoryIds.includes(event.category.id)
      })
      setFilteredEvents(filtered)
    } else {
      setFilteredEvents(allEvents)
    }

    setSelectedCategoryIds(newCategoryIds)
  }

  const value = {
    selectedDate,
    setSelectedDate: handleSelectDate,
    selectedUserId,
    setSelectedUserId,
    badgeVariant,
    setBadgeVariant,
    users,
    selectedColors,
    filterEventsBySelectedColors,
    filterEventsBySelectedUser,
    events: filteredEvents,
    view: currentView,
    use24HourFormat,
    toggleTimeFormat,
    setView,
    agendaModeGroupBy,
    setAgendaModeGroupBy,
    addEvent,
    updateEvent,
    removeEvent,
    clearFilter,
    categories,
    selectedCategoryIds,
    filterEventsByCategory
  }

  return <CalendarContext.Provider value={value}>{children}</CalendarContext.Provider>
}
