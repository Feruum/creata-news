import { PlatformRole, type User } from '@studyverse/proto'
import type { ColumnDef, FilterFn } from '@tanstack/react-table'
import { SendIcon } from 'lucide-react'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { formatDate } from '@/shared/lib/utils'
import { Badge } from '@/shared/ui/components/badge'

import { AssignRoleDialog } from './components/assign-role-dialog'
import { UserAvatar } from './components/user-avatar'

// Custom filter function for multi-column searching
export const multiColumnFilterFn: FilterFn<User> = (row, _columnId, filterValue) => {
  const { firstName, lastName, username, email } = row.original
  const fullName = firstName && lastName ? `${firstName} ${lastName}` : ''
  const searchableRowContent = `${fullName} ${username} ${email}`.toLowerCase()
  const searchTerm = (filterValue ?? '').toLowerCase()
  return searchableRowContent.includes(searchTerm)
}

// Helper to get display name
function getDisplayName(user: User): { name: string; isPending: boolean } {
  if (user.firstName && user.lastName) {
    return { name: `${user.firstName} ${user.lastName}`, isPending: false }
  }
  if (user.firstName) {
    return { name: user.firstName, isPending: false }
  }
  // No name set - show username but mark as pending
  return { name: user.username, isPending: true }
}

// Helper to get role badge styling
function getRoleBadge(role: PlatformRole) {
  switch (role) {
    case PlatformRole.ADMIN:
      return <Badge variant='default'>Admin</Badge>
    case PlatformRole.STAFF:
      return <Badge variant='secondary'>Staff</Badge>
    default:
      return <span className='text-xs text-muted-foreground'>User</span>
  }
}

// Column definitions - no select/delete columns (DataTable adds them automatically)
export const columns: ColumnDef<User>[] = [
  {
    accessorKey: 'id',
    header: 'ID',
    size: 60,
    minSize: 60,
    maxSize: 60,
    enableSorting: true,
    cell: ({ row }) => <span className='font-mono text-xs text-muted-foreground'>{row.original.id}</span>
  },
  {
    id: 'name',
    accessorKey: 'firstName',
    header: 'Name',
    size: 200,
    minSize: 150,
    enableSorting: true,
    filterFn: multiColumnFilterFn,
    cell: ({ row }) => {
      const { name, isPending } = getDisplayName(row.original)
      return (
        <div className='flex items-center gap-2 min-w-0'>
          <UserAvatar
            userId={row.original.id}
            username={row.original.username}
            name={name}
            avatarUrl={row.original.avatarUrl}
            telegramUsername={row.original.telegramUsername}
          />
          <div className='flex flex-col min-w-0'>
            <span className='flex items-center gap-1.5'>
              <span className='font-medium text-sm truncate' title={name}>
                {name}
              </span>
              <AchievementNameBadge badge={row.original.selectedBadge} userId={row.original.id} />
            </span>
            {isPending && <span className='text-xs text-muted-foreground'>Profile incomplete</span>}
          </div>
        </div>
      )
    }
  },
  {
    accessorKey: 'email',
    header: 'Email',
    size: 220,
    minSize: 150,
    enableSorting: true,
    cell: ({ row }) => (
      <span className='text-sm truncate' title={row.getValue('email') as string}>
        {row.getValue('email') as string}
      </span>
    )
  },
  {
    accessorKey: 'platformRole',
    header: 'Role',
    size: 100,
    minSize: 80,
    enableSorting: true,
    cell: ({ row }) => getRoleBadge(row.original.platformRole)
  },
  {
    accessorKey: 'telegramUsername',
    header: 'Telegram',
    size: 120,
    minSize: 100,
    enableSorting: true,
    cell: ({ row }) => {
      const telegramUsername = row.original.telegramUsername
      if (!telegramUsername) {
        return <span className='text-xs text-muted-foreground'>—</span>
      }
      return (
        <a
          href={`https://t.me/${telegramUsername}`}
          target='_blank'
          rel='noopener noreferrer'
          className='flex items-center gap-1 text-sm text-[#26A5E4] hover:underline'
          onClick={(event) => event.stopPropagation()}
        >
          <SendIcon className='h-4 w-4' />
          <span className='truncate'>@{telegramUsername}</span>
        </a>
      )
    }
  },
  {
    accessorKey: 'createdAt',
    header: 'Created',
    size: 120,
    minSize: 100,
    enableSorting: true,
    cell: ({ row }) => (
      <span className='text-xs text-muted-foreground tabular-nums whitespace-nowrap'>
        {formatDate(row.original.createdAt)}
      </span>
    )
  },
  {
    id: 'actions',
    header: '',
    size: 140,
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => <AssignRoleDialog user={row.original} />
  }
]
