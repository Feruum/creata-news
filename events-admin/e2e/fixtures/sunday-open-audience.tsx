import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { StatisticsService, UsersService } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createRoot } from 'react-dom/client'
import { SundayOpenAudienceStatistics } from '@/features/statistics'
import { AuthProvider } from '@/shared/lib/auth'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
let attempts = 0
const transport = createRouterTransport(({ service }) => {
  service(UsersService, {
    getMe: () => ({
      user: { id: 1, username: 'audience-fixture', email: 'staff@example.test' },
      permissions: { isStaff: !params.has('denied') }
    })
  })
  service(StatisticsService, {
    getSundayOpenAudienceStatistics: async () => {
      attempts++
      const counter = document.getElementById('rpc-count')
      if (counter) counter.textContent = String(attempts)
      if (params.has('serverDenied')) throw new ConnectError('Not permitted', Code.PermissionDenied)
      if (params.has('error') && attempts === 1) throw new ConnectError('Unavailable', Code.Unavailable)
      await new Promise((resolve) => setTimeout(resolve, 100))
      const empty = params.has('empty')
      return {
        totalAccounts: 8,
        completedAccounts: empty ? 0 : 4,
        roles: empty
          ? []
          : [
              { key: 'parent', count: 3 },
              { key: 'student', count: 1 }
            ],
        interests: empty
          ? []
          : [
              { key: 'it_ai', count: 3 },
              { key: 'design', count: 3 }
            ],
        goals: empty
          ? []
          : [
              { key: 'try_new', count: 4 },
              { key: 'meet_people', count: 2 }
            ],
        formats: empty
          ? []
          : [
              { key: 'workshops', count: 3 },
              { key: 'talks', count: 2 }
            ],
        sources: empty
          ? []
          : [
              { key: 'instagram', count: 3 },
              { key: 'friends', count: 1 }
            ]
      }
    }
  })
})
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={queryClient}>
    <TransportProvider transport={transport}>
      <AuthProvider>
        <main className='mx-auto max-w-6xl p-4'>
          <SundayOpenAudienceStatistics />
          <output id='rpc-count' aria-label='Aggregate requests'>
            0
          </output>
        </main>
      </AuthProvider>
    </TransportProvider>
  </QueryClientProvider>
)
