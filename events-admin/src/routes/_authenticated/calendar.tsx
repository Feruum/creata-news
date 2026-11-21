import { createFileRoute } from '@tanstack/react-router'
import { Calendar } from '@/features/calendar'

export const Route = createFileRoute('/_authenticated/calendar')({
  component: CalendarPage
})

function CalendarPage() {
  return <Calendar />
}
