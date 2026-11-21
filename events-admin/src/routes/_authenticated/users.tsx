import { createFileRoute } from '@tanstack/react-router'
import type { RowSelectionState } from '@tanstack/react-table'
import { SendIcon, UserPlusIcon } from 'lucide-react'
import { Suspense, useCallback, useState } from 'react'
import {
  columns,
  PreRegisterUserDialog,
  SendNotificationDialog,
  UserProfileModalControlled,
  useUsers
} from '@/features/users'
import { DataTable, useDataTableState } from '@/shared/components/admin-data-table'
import { Button } from '@/shared/ui/components/button'
import { Skeleton } from '@/shared/ui/components/skeleton'

export const Route = createFileRoute('/_authenticated/users')({
  component: () => (
    <Suspense fallback={<UsersLoading />}>
      <Users />
    </Suspense>
  )
})

const usersLoadingRowIds = [
  'user-row-1',
  'user-row-2',
  'user-row-3',
  'user-row-4',
  'user-row-5',
  'user-row-6',
  'user-row-7',
  'user-row-8',
  'user-row-9',
  'user-row-10'
]

function UsersLoading() {
  return (
    <div className='space-y-4'>
      {/* Header */}
      <div className='mb-6 space-y-2'>
        <Skeleton className='h-8 w-24' />
        <Skeleton className='h-5 w-48' />
      </div>

      {/* Toolbar */}
      <div className='mb-4 flex items-center justify-between gap-3'>
        <Skeleton className='h-10 w-64' />
        <Skeleton className='h-10 w-24' />
      </div>

      {/* Table */}
      <div className='rounded-lg border'>
        <div className='border-b px-4 py-3'>
          <Skeleton className='h-4 w-full max-w-2xl' />
        </div>
        {usersLoadingRowIds.map((rowId) => (
          <div key={rowId} className='border-b px-4 py-3 last:border-0'>
            <Skeleton className='h-5 w-full' />
          </div>
        ))}
      </div>

      {/* Pagination */}
      <div className='mt-4 flex items-center justify-between'>
        <Skeleton className='h-9 w-28' />
        <Skeleton className='h-5 w-40' />
        <Skeleton className='h-9 w-24' />
      </div>
    </div>
  )
}

function Users() {
  const { tableState } = useDataTableState({ defaultSortBy: 'id', defaultSortDesc: false })
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)

  const hasActiveFilters = Boolean(tableState.search)

  const { data } = useUsers({
    limit: hasActiveFilters ? 1000 : tableState.pageSize,
    page: hasActiveFilters ? 1 : tableState.page
  })

  const selectedUserIds = Object.keys(rowSelection).map((id) => parseInt(id, 10))

  const handleRowSelectionChange = useCallback(
    (updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState)) => {
      setRowSelection(typeof updater === 'function' ? updater(rowSelection) : updater)
    },
    [rowSelection]
  )

  const handleNotificationSuccess = useCallback(() => {
    setRowSelection({})
  }, [])

  const handleRowClick = useCallback((user: { id?: string | number }) => {
    if (typeof user.id === 'number') {
      setSelectedUserId(user.id)
    }
  }, [])

  return (
    <div className='space-y-6'>
      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Users</h1>
        <p className='text-sm text-muted-foreground mt-1'>Manage users, roles, and pre-registrations</p>
      </div>

      <DataTable
        data={data.users}
        columns={columns}
        totalCount={data.total || 0}
        hasActiveFilters={hasActiveFilters}
        searchColumnId='name'
        searchPlaceholder='Search users...'
        entityName='users'
        defaultSortBy='id'
        defaultSortDesc={false}
        rowSelection={rowSelection}
        onRowSelectionChange={handleRowSelectionChange}
        onRowClick={handleRowClick}
        toolbarActions={
          <div className='flex items-center gap-2'>
            {selectedUserIds.length > 0 && (
              <SendNotificationDialog userIds={selectedUserIds} onSuccess={handleNotificationSuccess}>
                <Button variant='outline'>
                  <SendIcon aria-hidden='true' className='-ms-1 opacity-60' size={16} />
                  Notify ({selectedUserIds.length})
                </Button>
              </SendNotificationDialog>
            )}
            <PreRegisterUserDialog>
              <Button className='ml-auto'>
                <UserPlusIcon aria-hidden='true' className='-ms-1 opacity-60' size={16} />
                Pre-register user
              </Button>
            </PreRegisterUserDialog>
          </div>
        }
      />

      <UserProfileModalControlled userId={selectedUserId} onClose={() => setSelectedUserId(null)} />
    </div>
  )
}
