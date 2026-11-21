import { useQuery } from '@connectrpc/connect-query'
import { AttendanceStatus, listUsers, type NameBadge, RegistrationStatus, type User } from '@studyverse/proto'
import { format } from 'date-fns'
import { ArrowDown, ArrowUp, ArrowUpDown, Loader2, Users } from 'lucide-react'
import { Suspense, useMemo, useState } from 'react'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/components/tabs'
import { cn } from '@/shared/ui/lib/utils'
import { useEventAttendance, useEventRegistrations } from '../api/queries'
import { ExternalRegistrationsTable } from './external-registrations-table'

type SortField =
  | 'attendee'
  | 'username'
  | 'email'
  | 'registration-status'
  | 'attendance-status'
  | 'registered-at'
  | 'check-in-at'

type SortDirection = 'asc' | 'desc'

interface RegistrationsTableRow {
  registrationId: number
  userId: number
  displayName: string
  selectedBadge?: NameBadge
  username: string
  email: string
  registrationStatus: RegistrationStatus
  attendanceStatus?: AttendanceStatus
  registeredAt?: string
  checkedInAt?: string
}

function RegistrationStatusBadge({ status }: { status: RegistrationStatus }) {
  switch (status) {
    case RegistrationStatus.REGISTERED:
      return (
        <Badge variant='secondary' className='bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400'>
          Registered
        </Badge>
      )
    case RegistrationStatus.CANCELLED:
      return (
        <Badge variant='secondary' className='bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400'>
          Cancelled
        </Badge>
      )
    case RegistrationStatus.WAITLIST:
      return (
        <Badge variant='secondary' className='bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-400'>
          Waitlisted
        </Badge>
      )
    case RegistrationStatus.PENDING_APPROVAL:
      return <Badge variant='secondary'>Pending approval</Badge>
    case RegistrationStatus.REJECTED:
      return <Badge variant='secondary'>Rejected</Badge>
    default:
      return <Badge variant='secondary'>Unknown</Badge>
  }
}

function AttendanceStatusBadge({ status }: { status: AttendanceStatus }) {
  switch (status) {
    case AttendanceStatus.CHECKED_IN:
      return (
        <Badge variant='secondary' className='bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400'>
          Checked In
        </Badge>
      )
    case AttendanceStatus.ATTENDED:
      return (
        <Badge
          variant='secondary'
          className='bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400'
        >
          Attended
        </Badge>
      )
    case AttendanceStatus.NO_SHOW:
      return (
        <Badge variant='secondary' className='bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300'>
          No Show
        </Badge>
      )
    default:
      return <Badge variant='secondary'>Unknown</Badge>
  }
}

function getDisplayName(user: User | undefined, fallbackUserId: number): string {
  if (!user) return `User #${fallbackUserId}`
  if (user.firstName && user.lastName) return `${user.firstName} ${user.lastName}`
  if (user.firstName) return user.firstName
  return user.username
}

function formatDateTime(value?: string): string {
  if (!value) return '—'
  return format(new Date(value), 'MMM d, yyyy HH:mm')
}

function getRegistrationStatusRank(status: RegistrationStatus): number {
  switch (status) {
    case RegistrationStatus.REGISTERED:
      return 5
    case RegistrationStatus.PENDING_APPROVAL:
      return 4
    case RegistrationStatus.REJECTED:
      return 3
    case RegistrationStatus.WAITLIST:
      return 2
    case RegistrationStatus.CANCELLED:
      return 1
    default:
      return 0
  }
}

function getAttendanceStatusRank(status?: AttendanceStatus): number {
  if (status === undefined || status === null) return 0
  switch (status) {
    case AttendanceStatus.ATTENDED:
      return 3
    case AttendanceStatus.CHECKED_IN:
      return 2
    case AttendanceStatus.NO_SHOW:
      return 1
    default:
      return 0
  }
}

function compareText(left: string, right: string): number {
  return left.localeCompare(right, 'en', { sensitivity: 'base' })
}

function compareNumber(left: number, right: number): number {
  return left - right
}

function toTimestamp(value?: string): number {
  if (!value) return 0
  const result = new Date(value).getTime()
  return Number.isFinite(result) ? result : 0
}

function RegistrationsTable({ eventId, isCheckInActive }: EventRegistrationsSectionProps) {
  const { data: registrationsData } = useEventRegistrations({ eventId, page: 1, limit: 200 }, isCheckInActive)
  const {
    data: attendanceData,
    isLoading: isAttendanceLoading,
    isError: isAttendanceError
  } = useEventAttendance({ eventId }, isCheckInActive)
  const { isAdmin, isStaff } = useCurrentUser()
  const canViewUserDirectory = isAdmin || isStaff
  // Event managers may view registrations without platform-wide user access.
  // Keep the event-scoped list usable when the global directory denies access.
  const { data: usersData, isError: isUsersError } = useQuery(
    listUsers,
    { page: 1, limit: 5000 },
    { enabled: canViewUserDirectory, retry: false }
  )

  const [sorting, setSorting] = useState<{ field: SortField; direction: SortDirection }>({
    field: 'registered-at',
    direction: 'desc'
  })

  const registrations = registrationsData.registrations

  const usersById = useMemo(() => {
    return new Map((usersData?.users || []).map((user) => [user.id, user]))
  }, [usersData?.users])

  const attendanceByRegistrationId = useMemo(() => {
    return new Map((attendanceData?.attendance || []).map((entry) => [entry.registrationId, entry]))
  }, [attendanceData?.attendance])

  const registrationCounters = useMemo(() => {
    return registrations.reduce(
      (acc, registration) => {
        if (registration.status === RegistrationStatus.CANCELLED) acc.cancelled += 1
        if (registration.status === RegistrationStatus.WAITLIST) acc.waitlist += 1
        return acc
      },
      { cancelled: 0, waitlist: 0 }
    )
  }, [registrations])

  const rows = useMemo<RegistrationsTableRow[]>(() => {
    return registrations.map((registration) => {
      const user = usersById.get(registration.userId)
      const attendance = attendanceByRegistrationId.get(registration.id)

      return {
        registrationId: registration.id,
        userId: registration.userId,
        displayName: getDisplayName(user, registration.userId),
        selectedBadge: user?.selectedBadge,
        username: user?.username || '',
        email: user?.email || '',
        registrationStatus: registration.status,
        attendanceStatus: attendance?.status,
        registeredAt: registration.registeredAt,
        checkedInAt: attendance?.checkedInAt
      }
    })
  }, [registrations, usersById, attendanceByRegistrationId])

  const sortedRows = useMemo(() => {
    const sorted = [...rows]

    sorted.sort((left, right) => {
      let result = 0

      switch (sorting.field) {
        case 'attendee':
          result = compareText(left.displayName, right.displayName)
          break
        case 'username':
          result = compareText(left.username, right.username)
          break
        case 'email':
          result = compareText(left.email, right.email)
          break
        case 'registration-status':
          result = compareNumber(
            getRegistrationStatusRank(left.registrationStatus),
            getRegistrationStatusRank(right.registrationStatus)
          )
          break
        case 'attendance-status':
          result = compareNumber(
            getAttendanceStatusRank(left.attendanceStatus),
            getAttendanceStatusRank(right.attendanceStatus)
          )
          break
        case 'registered-at':
          result = compareNumber(toTimestamp(left.registeredAt), toTimestamp(right.registeredAt))
          break
        case 'check-in-at':
          result = compareNumber(toTimestamp(left.checkedInAt), toTimestamp(right.checkedInAt))
          break
      }

      return sorting.direction === 'asc' ? result : result * -1
    })

    return sorted
  }, [rows, sorting.direction, sorting.field])

  const toggleSort = (field: SortField) => {
    setSorting((prev) => {
      if (prev.field === field) {
        return { field, direction: prev.direction === 'asc' ? 'desc' : 'asc' }
      }

      const defaultDirection = field === 'registered-at' || field === 'check-in-at' ? 'desc' : 'asc'
      return { field, direction: defaultDirection }
    })
  }

  const renderSortIcon = (field: SortField) => {
    if (sorting.field !== field) return <ArrowUpDown className='size-3.5 opacity-60' />
    return sorting.direction === 'asc' ? <ArrowUp className='size-3.5' /> : <ArrowDown className='size-3.5' />
  }

  const sortableHeader = (field: SortField, label: string) => {
    return (
      <Button
        variant='ghost'
        size='sm'
        className={cn('-ml-2 h-8 px-2 font-medium', sorting.field !== field && 'text-muted-foreground')}
        onClick={() => toggleSort(field)}
      >
        {label}
        {renderSortIcon(field)}
      </Button>
    )
  }

  if (!registrations.length) {
    return (
      <div className='flex flex-col items-center justify-center py-8 text-muted-foreground'>
        <Users className='mb-2 h-8 w-8 opacity-40' />
        <p className='text-sm'>No registrations yet</p>
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <Badge variant='outline'>Total: {registrationsData.total}</Badge>
        <Badge variant='outline'>Registered: {attendanceData?.totalRegistered ?? '—'}</Badge>
        <Badge variant='outline'>Cancelled: {registrationCounters.cancelled}</Badge>
        <Badge variant='outline'>Waitlist: {registrationCounters.waitlist}</Badge>
        <Badge variant='outline'>Checked In: {attendanceData?.totalCheckedIn ?? '—'}</Badge>
        <Badge variant='outline'>
          Attended: {attendanceData ? attendanceData.totalAttended - attendanceData.totalCheckedIn : '—'}
        </Badge>
        <Badge variant='outline'>No Show: {attendanceData?.totalNoShow ?? '—'}</Badge>
      </div>

      {isAttendanceError && (
        <p className='text-xs text-amber-600 dark:text-amber-400'>
          Attendance details are unavailable for this event with current permissions.
        </p>
      )}

      {(!canViewUserDirectory || isUsersError) && (
        <p className='text-xs text-muted-foreground'>
          These registrations came through EMS accounts. Profile details are restricted, so AITU affiliation is not
          confirmed here.
        </p>
      )}

      <div className='rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{sortableHeader('attendee', 'Attendee')}</TableHead>
              <TableHead>{sortableHeader('username', 'Username')}</TableHead>
              <TableHead>{sortableHeader('email', 'Email')}</TableHead>
              <TableHead>{sortableHeader('registration-status', 'Registration')}</TableHead>
              <TableHead>{sortableHeader('attendance-status', 'Attendance')}</TableHead>
              <TableHead>{sortableHeader('registered-at', 'Registered At')}</TableHead>
              <TableHead>{sortableHeader('check-in-at', 'Check-In At')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {sortedRows.map((row) => (
              <TableRow key={row.registrationId}>
                <TableCell>
                  <div className='min-w-0'>
                    <p className='flex items-center gap-1.5 font-medium'>
                      <span className='truncate'>{row.displayName}</span>
                      <AchievementNameBadge badge={row.selectedBadge} userId={row.userId} />
                    </p>
                    <p className='font-mono text-xs text-muted-foreground'>ID #{row.userId}</p>
                  </div>
                </TableCell>
                <TableCell className='text-sm text-muted-foreground'>
                  {row.username ? `@${row.username}` : '—'}
                </TableCell>
                <TableCell className='text-sm text-muted-foreground'>{row.email || '—'}</TableCell>
                <TableCell>
                  <RegistrationStatusBadge status={row.registrationStatus} />
                </TableCell>
                <TableCell>
                  {isAttendanceLoading ? (
                    <span className='text-xs text-muted-foreground'>Loading...</span>
                  ) : row.attendanceStatus !== undefined ? (
                    <AttendanceStatusBadge status={row.attendanceStatus} />
                  ) : (
                    <span className='text-xs text-muted-foreground'>Not marked</span>
                  )}
                </TableCell>
                <TableCell className='text-sm text-muted-foreground'>{formatDateTime(row.registeredAt)}</TableCell>
                <TableCell className='text-sm text-muted-foreground'>{formatDateTime(row.checkedInAt)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function RegistrationsLoading() {
  return (
    <div className='flex items-center justify-center py-8'>
      <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
    </div>
  )
}

interface EventRegistrationsSectionProps {
  eventId: number
  isCheckInActive?: boolean
}

export function EventRegistrationsSection({ eventId, isCheckInActive }: EventRegistrationsSectionProps) {
  return (
    <div className='rounded-lg border bg-muted/30 p-4'>
      <h3 className='mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground'>Registrations</h3>
      <Tabs key={eventId} defaultValue='participants'>
        <TabsList aria-label='Registration channel' className='mb-2 max-w-full'>
          <TabsTrigger value='participants'>EMS accounts</TabsTrigger>
          <TabsTrigger value='external'>External visitors</TabsTrigger>
        </TabsList>
        <TabsContent value='participants' keepMounted>
          <Suspense fallback={<RegistrationsLoading />}>
            <RegistrationsTable eventId={eventId} isCheckInActive={isCheckInActive} />
          </Suspense>
        </TabsContent>
        <TabsContent value='external'>
          <ExternalRegistrationsTable key={eventId} eventId={eventId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
