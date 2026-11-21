import { TransportProvider } from '@connectrpc/connect-query'
import { CheckInPhase, type Event, EventFormat, EventStatus, LocationType } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider
} from '@tanstack/react-router'
import { useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Calendar } from '@/features/calendar'
import { CreateEventForm, SimpleEventCardList } from '@/features/events'
import { SiteHeader } from '@/shared/components/site-header'
import { transport } from '@/shared/lib/api'
import { AuthProvider } from '@/shared/lib/auth'
import { ImageUpload } from '@/shared/ui/components/file-upload'
import { SidebarInset, SidebarProvider } from '@/shared/ui/components/sidebar'
import { Toaster } from '@/shared/ui/components/sonner'
import { ThemeProvider } from '@/shared/ui/components/theme-provider'
import '@/index.css'

const description = JSON.stringify('A valid saved description from the campus map.')
const events = [EventStatus.DRAFT, EventStatus.CANCELLED].map(
  (status, index) =>
    ({
      id: index + 1,
      title: index ? 'Cancelled test event' : 'Past draft event',
      status,
      description,
      format: EventFormat.ONLINE,
      organizationId: 1,
      startTime: '2020-01-01T10:00:00Z',
      endTime: '2020-01-01T11:00:00Z',
      checkInPhase: CheckInPhase.UNSPECIFIED
    }) as Event
)

function EventsFixture() {
  const [image, setImage] = useState<File | null>(null)
  const [edit, setEdit] = useState(false)
  return (
    <main className='space-y-4 p-4'>
      <h1>Local regression QA</h1>
      <SimpleEventCardList events={events} />
      <button type='button' onClick={() => setEdit(true)}>
        Edit saved map event
      </button>
      <CreateEventForm
        open={edit}
        onOpenChange={setEdit}
        eventId={1}
        initialValues={{
          title: 'Saved map event',
          description,
          startDate: new Date('2026-10-05'),
          endDate: new Date('2026-10-05'),
          startTime: '10:00',
          endTime: '11:00',
          organizationIds: [1],
          locationType: LocationType.ONLINE,
          onlineUrl: 'https://example.com'
        }}
      />
      <h2>Image rotation</h2>
      <div className='max-w-sm'>
        <ImageUpload value={image} onChange={setImage} aspectRatio='free' />
      </div>
      <p role='status'>{image ? `Applied ${image.type}: ${image.size} bytes` : 'Choose an image to test rotation'}</p>
    </main>
  )
}
function Shell() {
  return (
    <SidebarProvider>
      <SidebarInset>
        <SiteHeader />
        <Outlet />
        <Toaster />
      </SidebarInset>
    </SidebarProvider>
  )
}
const root = createRootRoute({ component: Shell })
const eventsRoute = createRoute({ getParentRoute: () => root, path: '/events', component: EventsFixture })
const calendarRoute = createRoute({ getParentRoute: () => root, path: '/calendar', component: Calendar })
const router = createRouter({
  routeTree: root.addChildren([eventsRoute, calendarRoute]),
  history: createMemoryHistory({ initialEntries: ['/events'] })
})
createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={new QueryClient()}>
    <TransportProvider transport={transport}>
      <ThemeProvider>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </ThemeProvider>
    </TransportProvider>
  </QueryClientProvider>
)
