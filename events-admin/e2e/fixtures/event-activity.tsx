import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { StatisticsService } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createRoot } from 'react-dom/client'
import { ChartAreaInteractive } from '@/shared/components/chart-area-interactive'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
document.documentElement.classList.toggle('dark', params.has('dark'))
let failed = false
const transport = createRouterTransport(({ service }) => {
  service(StatisticsService, {
    getEventTrends: ({ days }) => {
      if (params.has('error') && !failed) {
        failed = true
        throw new ConnectError('Unavailable', Code.Unavailable)
      }
      if (params.has('empty')) return { trends: [] }
      const date = (daysAgo: number) => {
        const day = new Date()
        day.setUTCDate(day.getUTCDate() - daysAgo)
        return day.toISOString().slice(0, 10)
      }
      return {
        trends: [
          { date: date(1), eventCount: 2, registrationCount: 11 },
          ...(days > 7 ? [{ date: date(10), eventCount: 1, registrationCount: 4 }] : []),
          ...(days > 30 ? [{ date: date(40), eventCount: 3, registrationCount: 6 }] : [])
        ]
      }
    }
  })
})
const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
createRoot(document.getElementById('root')!).render(
  <TransportProvider transport={transport}>
    <QueryClientProvider client={client}>
      <main style={{ maxWidth: 900, padding: 16, margin: '0 auto' }}>
        <ChartAreaInteractive />
      </main>
    </QueryClientProvider>
  </TransportProvider>
)
