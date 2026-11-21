import { create } from '@bufbuild/protobuf'
import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import { PlatformRole, UserSchema, UsersService } from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { NuqsTestingAdapter } from 'nuqs/adapters/testing'
import { Suspense, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import { useUsers } from '@/features/users'
import { columns } from '@/features/users/columns'
import { DataTable } from '@/shared/components/admin-data-table'
import '@/index.css'

const params = new URLSearchParams(window.location.search)
let user = create(UserSchema, {
  id: 5,
  username: 'role-user',
  firstName: 'Role',
  lastName: 'User',
  email: 'role-user@example.test',
  platformRole: PlatformRole.USER,
  createdAt: '2026-10-09T12:00:00Z'
})
const transport = createRouterTransport(({ service }) => {
  service(UsersService, {
    listUsers: () => ({ users: [user], total: 1 }),
    assignPlatformRole: ({ userId, role }) => {
      if (params.has('denied')) throw new ConnectError('Platform administrator access required', Code.PermissionDenied)
      if (userId !== user.id) throw new ConnectError('Wrong user', Code.NotFound)
      user = create(UserSchema, { ...user, platformRole: role })
      return { user }
    }
  })
})
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })

function UsersTable() {
  const { data } = useUsers()
  const [openedUser, setOpenedUser] = useState<number | null>(null)
  const [deletedUsers, setDeletedUsers] = useState<(string | number)[]>([])

  return (
    <>
      <DataTable
        data={data.users}
        columns={columns}
        totalCount={data.total}
        searchColumnId='name'
        entityName='users'
        readOnly={params.has('readOnly')}
        onRowClick={(selectedUser) => setOpenedUser(selectedUser.id)}
        onDeleteSelected={params.has('deletion') ? setDeletedUsers : undefined}
      />
      <output data-testid='opened-user'>{openedUser}</output>
      <output data-testid='deleted-users'>{JSON.stringify(deletedUsers)}</output>
    </>
  )
}

createRoot(document.getElementById('root')!).render(
  <NuqsTestingAdapter>
    <QueryClientProvider client={queryClient}>
      <TransportProvider transport={transport}>
        <Suspense fallback={<p>Loading users</p>}>
          <UsersTable />
        </Suspense>
        <Toaster />
      </TransportProvider>
    </QueryClientProvider>
  </NuqsTestingAdapter>
)
