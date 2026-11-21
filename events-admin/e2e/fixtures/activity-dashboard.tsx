import { createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { ActivityService, StatisticsService } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { ActivityTimeline } from '@/features/activity/components/activity-timeline'
import { ActivityWidget } from '@/features/activity/components/activity-widget'
import { EventActivityHeatmap } from '@/features/statistics/components/github-contributions'
import { ThemeProvider } from '@/shared/ui/components/theme-provider'
import { useTheme } from '@/shared/ui/hooks/use-theme'
import '@/index.css'

const transport = createRouterTransport(({ service }) => {
  service(StatisticsService, {
    getEventActivityByYear: ({ year }) => ({
      activities: year === 2024 ? [] : [{ date: `${year}-10-28`, count: 9, level: 4 }],
      totalEvents: year === 2024 ? 0 : 9
    })
  })
  const entries = [
    { id: 1n, actorId: 7, actorSnapshot: '{"display_name":"Ada Initiator"}', action: 'event.updated' },
    { id: 2n, actorId: 8, actorSnapshot: '{}', action: 'event.published' },
    { id: 3n, actorSnapshot: '{}', action: 'event.created' }
  ].map((entry) => ({
    ...entry,
    entityType: 'event',
    entityId: 1,
    entitySnapshot: '{"title":"Campus workshop"}',
    createdAt: new Date().toISOString()
  }))
  service(ActivityService, {
    listActivityLog: () => ({ entries, total: 3 }),
    listEntityActivityLog: () => ({ entries, total: 3 })
  })
})
const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
function Fixture() {
  const { setTheme } = useTheme()
  return (
    <main className='mx-auto max-w-5xl space-y-4 p-4'>
      <button type='button' onClick={() => setTheme('dark')}>
        Dark theme
      </button>
      <EventActivityHeatmap />
      <ActivityWidget />
      <ActivityTimeline entityType='event' entityId={1} />
    </main>
  )
}
createRoot(document.getElementById('root')!).render(
  <TransportProvider transport={transport}>
    <QueryClientProvider client={client}>
      <ThemeProvider defaultTheme='light'>
        <Suspense fallback='Loading'>
          <Fixture />
        </Suspense>
      </ThemeProvider>
    </QueryClientProvider>
  </TransportProvider>
)
