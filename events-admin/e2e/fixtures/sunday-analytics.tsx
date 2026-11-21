import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { EventsService, OrganizationsService, StatisticsService, UsersService } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider
} from '@tanstack/react-router'
import { Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { Route as AnalyticsRoute } from '@/routes/_authenticated/analytics'
import { Route as DashboardRoute } from '@/routes/_authenticated/index'
import { AppSidebar } from '@/shared/components/app-sidebar'
import { PwaProvider } from '@/shared/components/pwa-controls'
import { SiteHeader } from '@/shared/components/site-header'
import { AuthProvider } from '@/shared/lib/auth'
import { SidebarProvider } from '@/shared/ui/components/sidebar'
import { ThemeProvider } from '@/shared/ui/components/theme-provider'
import '@/index.css'

const params = new URLSearchParams(location.search)
let requests = 0
let fail = params.has('error')
let attended = 3
window.addEventListener('check-in', () => {
  attended = 4
})
const transport = createRouterTransport(({ service }) => {
  service(UsersService, {
    getMe: () => ({
      user: { id: 1, username: 'analyst' },
      permissions: { isAdmin: params.has('admin'), isStaff: !params.has('regular') && !params.has('admin') }
    })
  })
  service(OrganizationsService, { listOrganizations: () => ({}) })
  service(EventsService, { listEventsForAdmin: () => ({}) })
  service(StatisticsService, {
    getSundayOpenAnalytics: async () => {
      requests++
      document.documentElement.dataset.analyticsRequests = String(requests)
      if (params.has('slow')) await new Promise((resolve) => setTimeout(resolve, 1500))
      if (fail) {
        fail = false
        throw new ConnectError('Test outage', Code.Unavailable)
      }
      if (params.has('denied')) throw new ConnectError('Test denied', Code.PermissionDenied)
      const empty = params.has('empty')
      return {
        totalEvents: empty ? 0 : 2,
        totalRegistrations: empty ? 0 : 6,
        emsRegistrations: empty ? 0 : 4,
        externalRegistrations: empty ? 0 : 2,
        totalAttendees: empty ? 0 : attended,
        weeks: Array.from({ length: 7 }, (_, i) => ({
          weekStart: `2026-10-${String(5 + i).padStart(2, '0')}`,
          registrations: empty ? 0 : i
        })),
        topEvents: empty ? [] : [{ eventId: 1, title: 'Live API workshop', registrations: 6 }]
      }
    },
    getSundayOpenAudienceStatistics: () => ({
      totalAccounts: 8,
      completedAccounts: 2,
      roles: [{ key: 'student', count: 2 }]
    }),
    getEventActivityByYear: () => ({}),
    getEventTrends: () => ({}),
    getUserEngagementLevels: () => ({}),
    getEventCategoriesDistributionByMonth: () => ({}),
    getTopPerformingClubs: () => ({}),
    getTopPerformingEvents: () => ({}),
    getLowRegistrationEvents: () => ({}),
    getOrganizationActivity: () => ({})
  })
})
const root = createRootRoute({
  component: () => (
    <SidebarProvider>
      <AppSidebar />
      <main className='min-w-0 flex-1'>
        <SiteHeader />
        <div className='p-4'>
          <Suspense fallback='Loading'>
            <Outlet />
          </Suspense>
        </div>
      </main>
    </SidebarProvider>
  )
})
const index = createRoute({
  getParentRoute: () => root,
  path: '/',
  component: params.has('dashboard') ? DashboardRoute.options.component : () => <p>Fixture home</p>
})
const analytics = createRoute({
  getParentRoute: () => root,
  path: '/analytics',
  component: AnalyticsRoute.options.component
})
const router = createRouter({
  routeTree: root.addChildren([index, analytics]),
  history: createMemoryHistory({ initialEntries: [params.has('home') || params.has('dashboard') ? '/' : '/analytics'] })
})
const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
createRoot(document.getElementById('root')!).render(
  <AuthProvider>
    <TransportProvider transport={transport}>
      <QueryClientProvider client={client}>
        <ThemeProvider defaultTheme='light'>
          <PwaProvider>
            <RouterProvider router={router} />
          </PwaProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </TransportProvider>
  </AuthProvider>
)
