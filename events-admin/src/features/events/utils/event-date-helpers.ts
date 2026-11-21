import { format, formatDistanceToNow, isSameDay, parseISO, startOfDay } from 'date-fns'

export function formatEventDate(startTime: string, endTime?: string): string {
  const start = parseISO(startTime)
  const end = endTime ? parseISO(endTime) : start

  if (isSameDay(start, end)) {
    return format(start, 'EEEE, MMMM d, yyyy')
  }

  if (start.getFullYear() === end.getFullYear()) {
    if (start.getMonth() === end.getMonth()) {
      return `${format(start, 'MMMM d')} – ${format(end, 'd, yyyy')}`
    }
    return `${format(start, 'MMMM d')} – ${format(end, 'MMMM d, yyyy')}`
  }
  return `${format(start, 'MMMM d, yyyy')} – ${format(end, 'MMMM d, yyyy')}`
}

export function formatEventTime(startTime: string, endTime?: string): string | null {
  const start = parseISO(startTime)
  const end = endTime ? parseISO(endTime) : null

  const startHours = start.getHours()
  const startMinutes = start.getMinutes()
  const endHours = end?.getHours() ?? 0
  const endMinutes = end?.getMinutes() ?? 0

  if (startHours === 0 && startMinutes === 0 && endHours === 23 && endMinutes === 59) {
    return null
  }

  const startFormatted = format(start, 'h:mm a')
  if (!end || isSameDay(start, end)) {
    const endFormatted = end ? format(end, 'h:mm a') : null
    return endFormatted ? `${startFormatted} – ${endFormatted}` : startFormatted
  }

  return startFormatted
}

export function getRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  return formatDistanceToNow(date, { addSuffix: true })
}

export function isEventCalendarDateDisabled(date: Date, startDate?: Date, now = new Date()): boolean {
  const day = startOfDay(date).getTime()
  return day < startOfDay(now).getTime() || (startDate !== undefined && day < startOfDay(startDate).getTime())
}
