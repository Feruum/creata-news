import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  CategoriesService,
  EventsService,
  EventStatus,
  LocationType,
  OrganizationsService,
  RoomsService
} from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import { CalendarPersistenceProvider } from '@/features/calendar/calendar-persistence-provider'
import { CalendarProvider } from '@/features/calendar/contexts/calendar-context'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import { DndProvider } from '@/features/calendar/contexts/dnd-context'
import { useDragDrop } from '@/features/calendar/contexts/use-drag-drop'
import { DroppableArea } from '@/features/calendar/dnd/droppable-area'
import { ResizableEvent } from '@/features/calendar/dnd/resizable-event'
import type { IEvent } from '@/features/calendar/interfaces'
import { CreateEventForm } from '@/features/events/components/create-event-form'
import '@/index.css'

const query = new URLSearchParams(window.location.search)
const requests: Array<{ status?: EventStatus; startTime?: string; endTime?: string }> = []
let createCount = 0
let bookingCount = 0
const workflow: string[] = []
let rejectUpdate = query.has('rejectUpdates')
const transport = createRouterTransport(({ service }) => {
  service(OrganizationsService, { listOrganizations: () => ({ organizations: [{ id: 1, title: 'Primary host' }] }) })
  service(CategoriesService, { listCategories: () => ({ categories: [] }) })
  service(RoomsService, {
    listRooms: () => ({ rooms: [] }),
    createBookingRequest: () => {
      bookingCount++
      workflow.push('booking')
      window.dispatchEvent(new Event('requests-changed'))
      if (query.has('failBookingOnce') && bookingCount === 1) throw new ConnectError('Please retry', Code.Unavailable)
      return { autoApproved: !query.has('pending') }
    }
  })
  service(EventsService, {
    createEvent: (input) => {
      requests.push({ status: input.status })
      workflow.push(input.status === EventStatus.DRAFT ? 'create:draft' : 'create:published')
      window.dispatchEvent(new Event('requests-changed'))
      createCount++
      if (query.has('failFirst') && createCount === 1) throw new ConnectError('Please retry', Code.Unavailable)
      if (query.has('campus') && input.status !== EventStatus.DRAFT)
        throw new ConnectError('Room approval required', Code.FailedPrecondition)
      return { event: { id: 9, title: input.title, slug: 'test-event' } }
    },
    getEventPublishReadiness: () => {
      workflow.push('readiness')
      window.dispatchEvent(new Event('requests-changed'))
      return { canPublishNow: bookingCount > 0 && !query.has('pending') }
    },
    publishEvent: () => {
      workflow.push('publish')
      window.dispatchEvent(new Event('requests-changed'))
      return { event: { id: 9, status: EventStatus.PUBLISHED } }
    },
    updateEvent: (input) => {
      requests.push({ startTime: input.startTime, endTime: input.endTime })
      window.dispatchEvent(new Event('requests-changed'))
      if (rejectUpdate) throw new ConnectError('Permission denied', Code.PermissionDenied)
      return { event: { id: input.id, startTime: input.startTime, endTime: input.endTime } }
    }
  })
})

const original: IEvent = {
  id: 9,
  title: 'Test event',
  description: '',
  color: 'blue',
  startDate: '2026-10-02T09:00:00.000Z',
  endDate: '2026-10-02T10:00:00.000Z',
  user: { id: '1', name: 'Host', picturePath: null }
}

function CalendarControls() {
  const calendar = useCalendar()
  const dnd = useDragDrop()
  const [status, setStatus] = useState('idle')
  const update = async (resize: boolean, fail: boolean) => {
    rejectUpdate = fail
    try {
      await calendar.updateEvent({
        ...calendar.events[0],
        startDate: resize ? calendar.events[0].startDate : '2026-10-03T09:00:00.000Z',
        endDate: resize ? '2026-10-03T11:00:00.000Z' : '2026-10-03T10:00:00.000Z'
      })
      setStatus('saved')
    } catch {
      setStatus('failed')
    }
  }
  return (
    <>
      <output data-testid='calendar-dates'>
        {calendar.events.map((event) => `${event.startDate}/${event.endDate}`).join(',')}
      </output>
      <output data-testid='calendar-status'>{status}</output>
      <button type='button' onClick={() => update(false, false)}>
        Move event
      </button>
      <button type='button' onClick={() => update(true, false)}>
        Resize event
      </button>
      <button type='button' onClick={() => update(false, true)}>
        Fail move
      </button>
      <button type='button' draggable onDragStart={() => dnd.startDrag(calendar.events[0])}>
        Drag event
      </button>
      <DroppableArea date={new Date('2026-10-03T09:00:00.000Z')}>Drop event here</DroppableArea>
      <div data-testid='resize-event' style={{ width: 180 }}>
        <ResizableEvent event={calendar.events[0]}>
          <div data-testid='resize-body' style={{
            height: (Date.parse(calendar.events[0].endDate) - Date.parse(calendar.events[0].startDate)) / 3600000 * 96 - 8,
            width: 180
          }}>Resizable event</div>
        </ResizableEvent>
      </div>
    </>
  )
}

function Harness() {
  const [open, setOpen] = useState(true)
  const [createdDraft, setCreatedDraft] = useState<number | null>(null)
  const [events, setEvents] = useState([original])
  const [, refresh] = useState(0)
  useEffect(() => {
    const refreshRequests = () => refresh((value) => value + 1)
    window.addEventListener('requests-changed', refreshRequests)
    return () => window.removeEventListener('requests-changed', refreshRequests)
  }, [])

  return (
    <>
      <output data-testid='requests'>{JSON.stringify(requests)}</output>
      <output data-testid='workflow'>{JSON.stringify(workflow)}</output>
      <output data-testid='created-draft'>{createdDraft}</output>
      {query.has('calendar') ? (
        <>
          <button type='button' onClick={() => setEvents([{
            ...original, startDate: '2026-10-04T09:00:00.000Z', endDate: '2026-10-04T11:00:00.000Z'
          }])}>
            Refetch events
          </button>
          <CalendarProvider events={events} users={[original.user]}>
            <CalendarPersistenceProvider>
              <DndProvider showConfirmation={false}>
                <CalendarControls />
              </DndProvider>
            </CalendarPersistenceProvider>
          </CalendarProvider>
        </>
      ) : (
        <>
          <button type='button' onClick={() => setOpen(true)}>
            Open event form
          </button>
          <Suspense fallback={<p>Loading form</p>}>
            <CreateEventForm
              open={open}
              onOpenChange={setOpen}
              onCreatedDraft={setCreatedDraft}
              initialValues={{
                title: 'Initial event title',
                description: JSON.stringify([{ type: 'p', children: [{ text: 'A long enough event description' }] }]),
                startDate: new Date('2026-10-05'),
                endDate: new Date('2026-10-05'),
                startTime: '12:00',
                endTime: '13:00',
                organizationIds: [1],
                locationType: query.has('campus') ? LocationType.CAMPUS_ROOM : LocationType.ONLINE,
                roomId: query.has('campus') ? 'C1.1.123' : undefined,
                onlineUrl: 'https://example.com/meeting',
                requestRoomBooking: query.has('campus')
              }}
            />
          </Suspense>
        </>
      )}
      <Toaster />
    </>
  )
}

createRoot(document.getElementById('root')!).render(
  <TransportProvider transport={transport}>
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}
    >
      <Harness />
    </QueryClientProvider>
  </TransportProvider>
)
