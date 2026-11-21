import { type BookingRequest, BookingRequestStatus } from '@studyverse/proto'
import { createFileRoute } from '@tanstack/react-router'
import { ClipboardListIcon, FilterIcon } from 'lucide-react'
import { Suspense, useCallback, useState } from 'react'
import {
  ApproveBookingDialog,
  BookingDetailsDialog,
  ConflictsPanel,
  createColumns,
  getBookingStatusBadgeStyle,
  RejectBookingDialog,
  useBookingRequests
} from '@/features/bookings'
import { DataTable, useDataTableState } from '@/shared/components/admin-data-table'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { Skeleton } from '@/shared/ui/components/skeleton'

export const Route = createFileRoute('/_authenticated/bookings')({
  component: () => (
    <Suspense fallback={<BookingsLoading />}>
      <Bookings />
    </Suspense>
  )
})

function BookingsLoading() {
  return (
    <div className='space-y-4'>
      <div className='mb-6 space-y-2'>
        <Skeleton className='h-8 w-40' />
        <Skeleton className='h-5 w-72' />
      </div>

      <div className='mb-4 flex items-center justify-between gap-3'>
        <Skeleton className='h-10 w-64' />
        <Skeleton className='h-10 w-32' />
      </div>

      <div className='rounded-lg border'>
        <div className='border-b px-4 py-3'>
          <Skeleton className='h-4 w-full max-w-2xl' />
        </div>
        {[
          'booking-loading-1',
          'booking-loading-2',
          'booking-loading-3',
          'booking-loading-4',
          'booking-loading-5',
          'booking-loading-6',
          'booking-loading-7',
          'booking-loading-8',
          'booking-loading-9',
          'booking-loading-10'
        ].map((key) => (
          <div key={key} className='border-b px-4 py-3 last:border-0'>
            <Skeleton className='h-5 w-full' />
          </div>
        ))}
      </div>

      <div className='mt-4 flex items-center justify-between'>
        <Skeleton className='h-9 w-28' />
        <Skeleton className='h-5 w-40' />
        <Skeleton className='h-9 w-24' />
      </div>
    </div>
  )
}

const STATUS_OPTIONS = [
  { value: BookingRequestStatus.PENDING, label: 'Pending' },
  { value: BookingRequestStatus.APPROVED, label: 'Approved' },
  { value: BookingRequestStatus.REJECTED, label: 'Rejected' },
  { value: BookingRequestStatus.CANCELLED, label: 'Cancelled' }
]

function Bookings() {
  const { tableState } = useDataTableState({ defaultSortBy: 'createdAt', defaultSortDesc: true })

  const [statusFilter, setStatusFilter] = useState<BookingRequestStatus[]>([BookingRequestStatus.PENDING])
  const [approveDialogOpen, setApproveDialogOpen] = useState(false)
  const [rejectDialogOpen, setRejectDialogOpen] = useState(false)
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false)
  const [selectedRequest, setSelectedRequest] = useState<BookingRequest | null>(null)

  const hasActiveFilters = Boolean(tableState.search) || statusFilter.length > 0

  const { data } = useBookingRequests({
    status: statusFilter.length === 1 ? statusFilter[0] : undefined,
    limit: hasActiveFilters ? 1000 : tableState.pageSize,
    page: hasActiveFilters ? 1 : tableState.page
  })

  const handleApprove = useCallback((request: BookingRequest) => {
    setSelectedRequest(request)
    setApproveDialogOpen(true)
  }, [])

  const handleReject = useCallback((request: BookingRequest) => {
    setSelectedRequest(request)
    setRejectDialogOpen(true)
  }, [])

  const handleViewDetails = useCallback((request: BookingRequest) => {
    setSelectedRequest(request)
    setDetailsDialogOpen(true)
  }, [])

  const handleStatusFilterChange = useCallback((status: BookingRequestStatus, checked: boolean) => {
    setStatusFilter((prev) => (checked ? [...prev, status] : prev.filter((s) => s !== status)))
  }, [])

  const columns = createColumns({
    onApprove: handleApprove,
    onReject: handleReject,
    onViewDetails: handleViewDetails
  })

  const pendingCount = data.bookingRequests.filter(
    (request: BookingRequest) => request.status === BookingRequestStatus.PENDING
  ).length
  const pendingBadgeStyle = getBookingStatusBadgeStyle(BookingRequestStatus.PENDING)

  return (
    <div className='space-y-6'>
      <div>
        <div className='flex items-center gap-3'>
          <h1 className='text-2xl font-semibold tracking-tight'>Room Bookings</h1>
          {pendingCount > 0 && (
            <Badge variant='outline' className={pendingBadgeStyle.className}>
              {pendingCount} pending
            </Badge>
          )}
        </div>
        <p className='text-sm text-muted-foreground mt-1'>Review and manage room booking requests</p>
      </div>

      <ConflictsPanel onSelectRequest={handleViewDetails} onApproveRequest={handleApprove} />

      <DataTable
        data={data.bookingRequests}
        columns={columns}
        totalCount={data.total || 0}
        hasActiveFilters={hasActiveFilters}
        onClearFilters={() => setStatusFilter([])}
        searchColumnId='title'
        searchPlaceholder='Search requests...'
        entityName='booking requests'
        defaultSortBy='createdAt'
        defaultSortDesc={true}
        onRowClick={handleViewDetails}
        getRowId={(row) => String(row.id)}
        toolbarActions={
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button variant='outline' className='gap-2' />}>
              <FilterIcon className='h-4 w-4' />
              Status
              {statusFilter.length > 0 && (
                <Badge variant='secondary' className='ml-1 h-5 px-1.5'>
                  {statusFilter.length}
                </Badge>
              )}
            </DropdownMenuTrigger>
            <DropdownMenuContent align='end' className='w-48'>
              <DropdownMenuGroup>
                <DropdownMenuLabel>Filter by Status</DropdownMenuLabel>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              {STATUS_OPTIONS.map((option) => (
                <DropdownMenuCheckboxItem
                  key={option.value}
                  checked={statusFilter.includes(option.value)}
                  onCheckedChange={(checked) => handleStatusFilterChange(option.value, checked)}
                >
                  {option.label}
                </DropdownMenuCheckboxItem>
              ))}
              {statusFilter.length > 0 && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuCheckboxItem checked={false} onCheckedChange={() => setStatusFilter([])}>
                    Clear all
                  </DropdownMenuCheckboxItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        }
      />

      {data.bookingRequests.length === 0 && !hasActiveFilters && (
        <div className='flex flex-col items-center justify-center py-12 text-center'>
          <ClipboardListIcon className='h-12 w-12 text-muted-foreground/50 mb-4' />
          <h3 className='font-medium text-lg'>No booking requests</h3>
          <p className='text-muted-foreground text-sm mt-1'>
            Booking requests will appear here when users request room reservations.
          </p>
        </div>
      )}

      <ApproveBookingDialog open={approveDialogOpen} onOpenChange={setApproveDialogOpen} request={selectedRequest} />

      <RejectBookingDialog open={rejectDialogOpen} onOpenChange={setRejectDialogOpen} request={selectedRequest} />

      <BookingDetailsDialog
        open={detailsDialogOpen}
        onOpenChange={setDetailsDialogOpen}
        request={selectedRequest}
        onApprove={() => {
          setDetailsDialogOpen(false)
          handleApprove(selectedRequest!)
        }}
        onReject={() => {
          setDetailsDialogOpen(false)
          handleReject(selectedRequest!)
        }}
      />
    </div>
  )
}
