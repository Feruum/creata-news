import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  EventAttendanceService,
  EventRegistrationsService,
  RegistrationStatus,
  sundayopenModels,
  sundayopenServices,
  UsersService
} from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { EventRegistrationsSection } from '@/features/events'
import { AuthProvider } from '@/shared/lib/auth'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
let externalAttempts = 0
const transport = createRouterTransport(({ service }) => {
  service(EventRegistrationsService, {
    getEventRegistrations: ({ eventId }) => ({
      total: 1,
      registrations: [{ id: 1, eventId, userId: 1, status: RegistrationStatus.REGISTERED }]
    })
  })
  service(EventAttendanceService, {
    getEventAttendance: () => ({ totalRegistered: 1 })
  })
  service(UsersService, {
    getMe: () => ({
      user: { id: 1, username: 'account-attendee', email: 'account@example.test' },
      permissions: { isAdmin: true }
    }),
    listUsers: () => ({ users: [{ id: 1, username: 'account-attendee', email: 'account@example.test' }] })
  })
  service(sundayopenServices.SundayOpenService, {
    getExternalRegistrations: async ({ eventId, page, limit }) => {
      externalAttempts += 1
      if (params.has('error') && externalAttempts === 1) {
        throw new ConnectError('External visitors are unavailable', Code.Unavailable)
      }
      if (params.has('denied')) throw new ConnectError('Not permitted', Code.PermissionDenied)
      await new Promise((resolve) => setTimeout(resolve, 150))
      const total = params.has('empty') ? 0 : 26
      const offset = ((page || 1) - 1) * (limit || 25)
      return {
        total,
        registrations: Array.from({ length: Math.min(limit || 25, Math.max(0, total - offset)) }, (_, index) => ({
          id: offset + index + 1,
          eventId,
          fullName: `Event ${eventId} visitor ${offset + index + 1}`,
          phone: params.has('missingPhone') ? undefined : '+77000000000',
          email: `visitor${offset + index + 1}@example.test`,
          status:
            index === 0
              ? sundayopenModels.ExternalRegistrationStatus.CANCELLED
              : sundayopenModels.ExternalRegistrationStatus.REGISTERED,
          registeredAt: '2026-10-19T12:00:00Z'
        }))
      }
    }
  })
})
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

function Harness() {
  const [eventId, setEventId] = useState(1)
  return (
    <QueryClientProvider client={queryClient}>
      <TransportProvider transport={transport}>
        <button type='button' onClick={() => setEventId((id) => id + 1)}>
          Next event
        </button>
        <AuthProvider>
          <EventRegistrationsSection eventId={eventId} />
        </AuthProvider>
      </TransportProvider>
    </QueryClientProvider>
  )
}

createRoot(document.getElementById('root')!).render(<Harness />)
