import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { CheckInPhase, EventStatus } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider
} from '@tanstack/react-router'
import { createRoot } from 'react-dom/client'
import { getEventStatus } from '@/features/events/utils/event-status'
import { SiteHeader } from '@/shared/components/site-header'
import { EventStatusBadge } from '@/shared/ui/components/event-status-badge'
import { Sidebar, SidebarContent, SidebarInset, SidebarProvider } from '@/shared/ui/components/sidebar'
import { ThemeProvider } from '@/shared/ui/components/theme-provider'
import '@/index.css'
const queryClient = new QueryClient()
const transport = createRouterTransport(() => {})
function Fixture() {
  return (
    <ThemeProvider>
      <SidebarProvider>
        <Sidebar>
          <SidebarContent>
            <p className='p-4'>QA navigation menu</p>
          </SidebarContent>
        </Sidebar>
        <SidebarInset>
          <SiteHeader />
          <main className='p-4'>
            <h1>Local EMS header QA</h1>
            <p>Past draft</p>
            <EventStatusBadge
              status={getEventStatus({
                status: EventStatus.DRAFT,
                checkInPhase: CheckInPhase.UNSPECIFIED,
                startTime: '2020-01-01T00:00:00Z',
                endTime: '2020-01-01T01:00:00Z'
              })}
            />
            <p>Future cancelled event</p>
            <EventStatusBadge
              status={getEventStatus({
                status: EventStatus.CANCELLED,
                checkInPhase: CheckInPhase.UNSPECIFIED,
                startTime: '2099-01-01T00:00:00Z',
                endTime: '2099-01-01T01:00:00Z'
              })}
            />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </ThemeProvider>
  )
}
const root = createRootRoute({ component: Outlet })
const route = createRoute({ getParentRoute: () => root, path: '/events', component: Fixture })
const router = createRouter({
  routeTree: root.addChildren([route]),
  history: createMemoryHistory({ initialEntries: ['/events'] })
})
createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <TransportProvider transport={transport}>
      <RouterProvider router={router} />
    </TransportProvider>
  </QueryClientProvider>
)
