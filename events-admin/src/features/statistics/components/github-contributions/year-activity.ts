import { eachDayOfInterval, format, parseISO } from 'date-fns'
import type { Activity } from '../kibo-ui/contribution-graph'

export function fillActivityYear(activities: Activity[], year: number): Activity[] {
  const byDate = new Map(activities.map((activity) => [activity.date, activity]))
  return eachDayOfInterval({ start: parseISO(`${year}-01-01`), end: parseISO(`${year}-12-31`) }).map((day) => {
    const date = format(day, 'yyyy-MM-dd')
    return byDate.get(date) ?? { date, count: 0, level: 0 }
  })
}
