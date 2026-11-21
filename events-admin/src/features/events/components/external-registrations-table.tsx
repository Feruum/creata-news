import { Code, ConnectError } from '@connectrpc/connect'
import { sundayopenModels } from '@studyverse/proto'
import { format } from 'date-fns'
import { Loader2, Users } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import { useExternalRegistrations } from '../api/queries'

const PAGE_SIZE = 25

function ExternalStatusBadge({ status }: { status: sundayopenModels.ExternalRegistrationStatus }) {
  switch (status) {
    case sundayopenModels.ExternalRegistrationStatus.REGISTERED:
      return (
        <Badge variant='secondary' className='bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400'>
          Registered
        </Badge>
      )
    case sundayopenModels.ExternalRegistrationStatus.CANCELLED:
      return (
        <Badge variant='secondary' className='bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400'>
          Cancelled
        </Badge>
      )
    case sundayopenModels.ExternalRegistrationStatus.PENDING_APPROVAL:
      return <Badge variant='secondary'>Pending approval</Badge>
    case sundayopenModels.ExternalRegistrationStatus.REJECTED:
      return <Badge variant='secondary'>Rejected</Badge>
    default:
      return <Badge variant='secondary'>Unknown</Badge>
  }
}

function formatRegisteredAt(value: string) {
  const date = new Date(value)
  return Number.isFinite(date.getTime()) ? format(date, 'MMM d, yyyy HH:mm') : '—'
}

export function ExternalRegistrationsTable({ eventId }: { eventId: number }) {
  const [page, setPage] = useState(1)
  const { data, isPending, isFetching, isError, error, refetch } = useExternalRegistrations({
    eventId,
    page,
    limit: PAGE_SIZE
  })
  const totalPages = Math.max(1, Math.ceil((data?.total || 0) / PAGE_SIZE))
  const permissionDenied = error instanceof ConnectError && error.code === Code.PermissionDenied

  return (
    <div className='space-y-3'>
      <p className='text-xs text-muted-foreground'>
        Registrations made on the Sunday Open website. This list indicates the registration channel, not AITU
        affiliation.
      </p>
      {isPending ? (
        <div role='status' className='flex items-center justify-center gap-2 py-8 text-sm text-muted-foreground'>
          <Loader2 aria-hidden='true' className='size-5 animate-spin' />
          Loading external visitors...
        </div>
      ) : isError ? (
        <div className='space-y-2 py-4'>
          <p role='alert' className='text-sm text-destructive'>
            {permissionDenied
              ? 'You do not have permission to view external visitors.'
              : 'Unable to load external visitors. Please try again.'}
          </p>
          <Button variant='outline' size='sm' disabled={isFetching} onClick={() => void refetch()}>
            Retry
          </Button>
        </div>
      ) : (
        <>
          <Badge variant='outline'>Total: {data?.total ?? 0}</Badge>
          {!data?.registrations.length ? (
            <div className='flex flex-col items-center justify-center py-8 text-muted-foreground'>
              <Users aria-hidden='true' className='mb-2 size-8 opacity-40' />
              <p className='text-sm'>No external visitors yet</p>
            </div>
          ) : (
            <div className='rounded-md border'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Phone</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Registered At</TableHead>
                    <TableHead>Checked In At</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.registrations.map((registration) => (
                    <TableRow key={registration.id}>
                      <TableCell className='font-medium'>{registration.fullName || '—'}</TableCell>
                      <TableCell>{registration.phone || '—'}</TableCell>
                      <TableCell>{registration.email || '—'}</TableCell>
                      <TableCell>
                        <ExternalStatusBadge status={registration.status} />
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-muted-foreground'>
                        {formatRegisteredAt(registration.registeredAt)}
                      </TableCell>
                      <TableCell className='whitespace-nowrap text-muted-foreground'>
                        {registration.checkedInAt ? formatRegisteredAt(registration.checkedInAt) : '—'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </>
      )}
      {(page > 1 || (!isPending && !isError && totalPages > 1)) && (
        <nav aria-label='External visitors pagination' className='flex flex-wrap items-center justify-end gap-2'>
          <span aria-live='polite' className='mr-auto text-xs text-muted-foreground'>
            Page {page}
            {data && !isError ? ` of ${totalPages}` : ''}
          </span>
          <Button
            aria-label='Previous page'
            variant='outline'
            size='sm'
            disabled={page === 1 || isFetching}
            onClick={() => setPage((current) => current - 1)}
          >
            Previous
          </Button>
          <Button
            aria-label='Next page'
            variant='outline'
            size='sm'
            disabled={isPending || isError || page >= totalPages || isFetching}
            onClick={() => setPage((current) => current + 1)}
          >
            Next
          </Button>
        </nav>
      )}
    </div>
  )
}
