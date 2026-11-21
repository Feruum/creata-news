import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  AttendanceStatus,
  CategoriesService,
  CheckInPhase,
  EventAttendanceService,
  EventRegistrationsService,
  EventStatus,
  EventsService,
  OrganizationsService,
  RegistrationStatus,
  RoomsService,
  StatisticsService,
  UsersService
} from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { EventDetailsModalControlled } from '@/features/events'
import { EventAnalytics } from '@/shared/components/event-analytics'
import { AuthProvider } from '@/shared/lib/auth'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
const requests: Record<string, number> = {}
let checkedIn = params.has('many')
const activationTime = Date.now() + 10_000
window.addEventListener('external-check-in', () => {
  checkedIn = true
})

function requested(name: string) {
  requests[name] = (requests[name] || 0) + 1
  window.dispatchEvent(new Event('rpc-request'))
}

function totals() {
  return {
    totalRegistered: params.has('many') ? 500 : 1,
    totalAttended: params.has('many') ? 101 : Number(checkedIn),
    totalCheckedIn: params.has('many') ? 100 : Number(checkedIn),
    totalNoShow: params.has('many') ? 3 : 0
  }
}

function event() {
  return {
    id: 1,
    title: 'Attendance fixture',
    description: '[]',
    organization: { id: 1, title: 'Fixture organization' },
    organizationId: 1,
    userId: 1,
    startTime: '2026-10-11T11:00:00Z',
    endTime: '2026-10-11T13:00:00Z',
    status: EventStatus.PUBLISHED,
    checkInPhase:
      params.has('upcoming') && Date.now() < activationTime
        ? CheckInPhase.NONE
        : params.has('locked')
          ? CheckInPhase.LOCKED
          : params.has('grace')
            ? CheckInPhase.GRACE
            : CheckInPhase.LIVE,
    totalRegistrations: totals().totalRegistered,
    totalAttendees: totals().totalAttended
  }
}

const transport = createRouterTransport(({ service }) => {
  service(CategoriesService, { listCategories: () => ({ categories: [] }) })
  service(OrganizationsService, { listOrganizations: () => ({ organizations: [] }) })
  service(RoomsService, { listRooms: () => ({ rooms: [] }) })
  service(EventsService, {
    getEvent: () => {
      requested('event')
      return { event: event(), canManageRegistrations: true }
    },
    listEventsForAdmin: () => {
      requested('analytics')
      return { events: [event()], total: 1 }
    }
  })
  service(EventRegistrationsService, {
    getEventRegistrations: () => {
      requested('registrations')
      return {
        total: totals().totalRegistered,
        registrations: [{ id: 1, eventId: 1, userId: 1, status: RegistrationStatus.REGISTERED }]
      }
    }
  })
  service(EventAttendanceService, {
    getEventAttendance: () => {
      requested('attendance')
      return {
        ...totals(),
        attendance: checkedIn
          ? [{ id: 1, registrationId: 1, status: AttendanceStatus.CHECKED_IN, checkedInAt: '2026-10-11T12:00:00Z' }]
          : []
      }
    },
    getCheckInInfo: () => ({ checkInPhase: event().checkInPhase })
  })
  service(UsersService, {
    getMe: () => ({
      user: { id: 1, username: 'attendee', email: 'attendee@test.local' },
      permissions: { isAdmin: true }
    }),
    listUsers: () => ({ users: [{ id: 1, username: 'attendee', email: 'attendee@test.local' }] })
  })
  service(StatisticsService, {
    getTopPerformingEvents: () => ({ events: [] }),
    getLowRegistrationEvents: () => ({ events: [] }),
    getOrganizationActivity: () => ({ organizations: [] })
  })
})

const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: 300000, retry: false } } })

function Harness() {
  const [open, setOpen] = useState(true)
  const [counts, setCounts] = useState(requests)
  useEffect(() => {
    const sync = () => setCounts({ ...requests })
    window.addEventListener('rpc-request', sync)
    sync()
    return () => window.removeEventListener('rpc-request', sync)
  }, [])
  return (
    <QueryClientProvider client={queryClient}>
      <TransportProvider transport={transport}>
        <output data-testid='requests'>{JSON.stringify(counts)}</output>
        <button type='button' onClick={() => setOpen(true)}>
          Open event
        </button>
        <Suspense fallback={<p>Loading attendance fixture...</p>}>
          {params.has('analytics') ? (
            <EventAnalytics />
          ) : (
            <EventDetailsModalControlled eventId={open ? 1 : null} onClose={() => setOpen(false)} />
          )}
        </Suspense>
      </TransportProvider>
    </QueryClientProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <AuthProvider>
    <Harness />
  </AuthProvider>
)
