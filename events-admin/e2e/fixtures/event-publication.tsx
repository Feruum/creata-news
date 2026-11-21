import { create, toJson } from '@bufbuild/protobuf'
import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  CategoriesService,
  CreateEventRequestSchema,
  EventPublishReadinessState,
  EventSchema,
  EventStatus,
  EventsService,
  LocationType,
  OrganizationSchema,
  OrganizationsService,
  RoomsService,
  UpdateEventRequestSchema
} from '@studyverse/proto'
import { QueryClientProvider } from '@tanstack/react-query'
import { Suspense, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import { CreateEventForm, EventDetailsModalControlled } from '@/features/events'
import { queryClient } from '@/shared/lib/api'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
const edit = params.has('edit')
const organizations = [
  create(OrganizationSchema, { id: 1, title: 'Sunday Open', slug: 'sunday-open' }),
  create(OrganizationSchema, { id: 2, title: 'Campus club', slug: 'campus-club' })
]
const host = organizations.find((organization) => organization.slug === params.get('host')) ?? organizations[0]!
const targets = params.get('targets') ?? 'sunday'
const requests: Array<{ method: string; input: unknown }> = []
let bookingState = EventPublishReadinessState.ROOM_BOOKING_MISSING
let bookingCount = 0
window.addEventListener('approve-booking', () => {
  bookingState = EventPublishReadinessState.READY
})
let event = create(EventSchema, {
  id: 9,
  title: 'Sunday workshop',
  description: JSON.stringify([
    { type: 'p', children: [{ text: 'A complete workshop description for publication QA' }] }
  ]),
  organizationId: host.id,
  organization: host,
  startTime: params.has('seconds') ? '2027-10-21T12:00:37.654Z' : '2027-10-21T12:00:00Z',
  endTime: params.has('seconds') ? '2027-10-21T13:00:52.456Z' : '2027-10-21T13:00:00Z',
  status: EventStatus.PUBLISHED,
  structuredLocation: params.has('campus')
    ? { type: LocationType.CAMPUS_ROOM, roomId: 'С1.2.321', roomName: 'Original room', floor: 2 }
    : { type: LocationType.ONLINE, onlineUrl: 'https://example.test/workshop' },
  publishOnMap: targets === 'map' || targets === 'both',
  publishOnSundayOpen: targets === 'sunday' || targets === 'both'
})
const transport = createRouterTransport(({ service }) => {
  service(OrganizationsService, { listOrganizations: () => ({ organizations }) })
  service(CategoriesService, { listCategories: () => ({ categories: [] }) })
  service(RoomsService, {
    listRooms: () => ({ rooms: [{ id: 'C1.2.322', name: 'New review room', floor: 2 }] }),
    createBookingRequest: (input) => {
      bookingCount++
      requests.push({ method: 'booking', input: { ...input, $typeName: undefined } })
      window.dispatchEvent(new Event('requests-changed'))
      if (params.has('denyBooking')) throw new ConnectError('technical permission policy', Code.PermissionDenied)
      if (params.has('failBookingOnce') && bookingCount === 1)
        throw new ConnectError('booking unavailable', Code.Unavailable)
      bookingState = params.has('autoApprove')
        ? EventPublishReadinessState.READY
        : EventPublishReadinessState.WAITING_ROOM_APPROVAL
      return { autoApproved: params.has('autoApprove') }
    }
  })
  service(EventsService, {
    getEvent: () => ({ event, canEdit: true }),
    publishEvent: (input) => {
      requests.push({ method: 'publish', input: { eventId: input.eventId } })
      event = { ...event, status: EventStatus.PUBLISHED }
      window.dispatchEvent(new Event('requests-changed'))
      return { event }
    },
    uploadEventImage: async (input) => {
      requests.push({ method: 'upload', input: { eventId: input.eventId, bytes: input.image.length } })
      window.dispatchEvent(new Event('requests-changed'))
      if (params.has('slowUpload')) await new Promise((resolve) => setTimeout(resolve, 500))
      if (params.has('failUpload')) throw new ConnectError('storage internal failure', Code.Unavailable)
      return { publicUrl: 'https://example.test/cover.png' }
    },
    getEventPublishReadiness: async (input) => {
      requests.push({ method: 'readiness', input: { ...input, $typeName: undefined } })
      window.dispatchEvent(new Event('requests-changed'))
      if (params.has('slowReadiness')) await new Promise((resolve) => setTimeout(resolve, 500))
      if (
        input.proposedLocation?.type === LocationType.ONLINE ||
        input.proposedLocation?.type === LocationType.EXTERNAL
      ) {
        return { state: EventPublishReadinessState.NO_ROOM_APPROVAL_REQUIRED, canPublishNow: true }
      }
      return {
        state: bookingState,
        canPublishNow: bookingState === EventPublishReadinessState.READY,
        requiresRoomApproval: true
      }
    },
    createEvent: (input) => {
      requests.push({ method: 'create', input: toJson(CreateEventRequestSchema, input) })
      window.dispatchEvent(new Event('requests-changed'))
      if (params.has('failCreate')) throw new ConnectError('internal insert error', Code.Unavailable)
      const organization = organizations.find((candidate) => candidate.id === input.organizationId)!
      const isSundayOpen = organization.slug === 'sunday-open'
      event = {
        ...event,
        title: input.title,
        description: input.description,
        startTime: input.startTime,
        endTime: input.endTime,
        structuredLocation: input.structuredLocation,
        organizationId: organization.id,
        organization,
        status: input.status ?? EventStatus.PUBLISHED,
        publishOnMap: !isSundayOpen,
        publishOnSundayOpen: isSundayOpen
      }
      window.dispatchEvent(new Event('requests-changed'))
      return { event }
    },
    updateEvent: (input) => {
      requests.push({ method: 'update', input: toJson(UpdateEventRequestSchema, input) })
      window.dispatchEvent(new Event('requests-changed'))
      if (params.has('revokeOnSave')) {
        bookingState = EventPublishReadinessState.WAITING_ROOM_APPROVAL
        throw new ConnectError('internal booking approval changed', Code.FailedPrecondition)
      }
      if (params.has('failUpdate'))
        throw new ConnectError(
          'database room_booking_id=42',
          params.get('failUpdate') === 'permission' ? Code.PermissionDenied : Code.Unavailable
        )
      event = { ...event, title: input.title ?? event.title, description: input.description ?? event.description }
      window.dispatchEvent(new Event('requests-changed'))
      return { event }
    }
  })
})

function Harness() {
  const [open, setOpen] = useState(true)
  const [, refresh] = useState(0)
  useEffect(() => {
    const changed = () => refresh((value) => value + 1)
    window.addEventListener('requests-changed', changed)
    return () => window.removeEventListener('requests-changed', changed)
  }, [])
  return (
    <>
      <output data-testid='publication-requests'>{JSON.stringify(requests)}</output>
      <output data-testid='saved-publication'>
        {JSON.stringify({ publishOnMap: event.publishOnMap, publishOnSundayOpen: event.publishOnSundayOpen })}
      </output>
      <Suspense fallback={<p>Loading event</p>}>
        {edit ? (
          <EventDetailsModalControlled eventId={open ? 9 : null} onClose={() => setOpen(false)} />
        ) : (
          <CreateEventForm
            open={open}
            onOpenChange={setOpen}
            initialValues={{
              title: event.title,
              description: event.description,
              startDate: new Date('2027-10-21'),
              endDate: new Date('2027-10-21'),
              startTime: '12:00',
              endTime: '13:00',
              organizationIds: [host.id],
              locationType: params.has('campus') ? LocationType.CAMPUS_ROOM : LocationType.ONLINE,
              roomId: params.has('campus') ? 'C1.2.321' : undefined,
              onlineUrl: 'https://example.test/workshop',
              coverImage: params.has('emptyImage')
                ? new File([], 'empty.png', { type: 'image/png' })
                : params.has('upload')
                  ? new File(['cover'], 'cover.png', { type: 'image/png' })
                  : undefined
            }}
          />
        )}
      </Suspense>
      <Toaster />
    </>
  )
}

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <TransportProvider transport={transport}>
      <Harness />
    </TransportProvider>
  </QueryClientProvider>
)
