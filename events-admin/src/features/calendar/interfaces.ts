import type { TEventColor } from '@/features/calendar/types'

export interface IUser {
  id: string
  name: string
  picturePath: string | null
}

export interface ICategory {
  id: number
  name: string
  color: string
}

export interface IEvent {
  id: number
  startDate: string
  endDate: string
  title: string
  color: TEventColor
  hexColor?: string | null
  description: string
  user: IUser
  isAllDay?: boolean
  category?: ICategory | null
}

export interface ICalendarCell {
  day: number
  currentMonth: boolean
  date: Date
}
