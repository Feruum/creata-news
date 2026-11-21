import { type BookingRequest, BookingRequestStatus } from '@studyverse/proto'
import type { ColumnDef } from '@tanstack/react-table'
import { format, parseISO } from 'date-fns'
import {
  Building2Icon,
  CalendarIcon,
  CheckIcon,
  ClockIcon,
  MapPinIcon,
  MoreHorizontalIcon,
  UserIcon,
  XIcon
} from 'lucide-react'

import { getBookingPriorityBadgeStyle, getBookingStatusBadgeStyle } from '@/features/bookings/booking-badge-styles'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/shared/ui/components/tooltip'
import { cn } from '@/shared/ui/lib/utils'

function formatDateTime(isoString: string): { date: string; time: string } {
  try {
    const date = parseISO(isoString)
    return {
      date: format(date, 'MMM d, yyyy'),
      time: format(date, 'HH:mm')
    }
  } catch {
    return { date: 'Invalid', time: '' }
  }
}

interface CreateColumnsOptions {
  onApprove: (request: BookingRequest) => void
  onReject: (request: BookingRequest) => void
  onViewDetails: (request: BookingRequest) => void
}

export function createColumns({
  onApprove,
  onReject,
  onViewDetails
}: CreateColumnsOptions): ColumnDef<BookingRequest>[] {
  return [
    {
      accessorKey: 'title',
      header: 'Request',
      cell: ({ row }) => {
        const request = row.original
        const priorityConfig = getBookingPriorityBadgeStyle(request.priority)

        return (
          <div className='flex flex-col gap-1 min-w-0'>
            <div className='flex items-center gap-2'>
              <span className='font-medium truncate'>{request.title}</span>
              <Badge variant='outline' className={cn('text-[10px] shrink-0', priorityConfig.className)}>
                {priorityConfig.label}
              </Badge>
            </div>
            {request.description && (
              <span className='text-xs text-muted-foreground truncate max-w-[300px]'>{request.description}</span>
            )}
          </div>
        )
      },
      filterFn: (row, _columnId, filterValue: string) => {
        const title = row.original.title?.toLowerCase() || ''
        const description = row.original.description?.toLowerCase() || ''
        const search = filterValue.toLowerCase()
        return title.includes(search) || description.includes(search)
      }
    },
    {
      accessorKey: 'room',
      header: 'Room',
      cell: ({ row }) => {
        const room = row.original.room
        if (!room) return <span className='text-muted-foreground'>-</span>

        return (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger render={<div className='flex items-center gap-2' />}>
                <MapPinIcon className='h-4 w-4 text-muted-foreground shrink-0' />
                <span className='truncate max-w-[150px]'>{room.name}</span>
              </TooltipTrigger>
              <TooltipContent>
                <p>
                  {room.block} • Room {room.roomNumber} • Floor {room.floor}
                </p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )
      }
    },
    {
      accessorKey: 'startTime',
      header: 'Date & Time',
      cell: ({ row }) => {
        const start = formatDateTime(row.original.startTime)
        const end = formatDateTime(row.original.endTime)

        return (
          <div className='flex flex-col gap-0.5 text-sm'>
            <div className='flex items-center gap-1.5'>
              <CalendarIcon className='h-3.5 w-3.5 text-muted-foreground' />
              <span>{start.date}</span>
            </div>
            <div className='flex items-center gap-1.5 text-muted-foreground'>
              <ClockIcon className='h-3.5 w-3.5' />
              <span>
                {start.time} - {end.time}
              </span>
            </div>
          </div>
        )
      },
      sortingFn: (rowA, rowB) => {
        return new Date(rowA.original.startTime).getTime() - new Date(rowB.original.startTime).getTime()
      }
    },
    {
      accessorKey: 'requesterName',
      header: 'Requester',
      cell: ({ row }) => {
        const request = row.original

        return (
          <div className='flex flex-col gap-0.5'>
            <div className='flex items-center gap-1.5'>
              <UserIcon className='h-3.5 w-3.5 text-muted-foreground' />
              <span className='truncate max-w-[120px]'>{request.requesterName || 'Unknown'}</span>
            </div>
            {request.organizationName && (
              <div className='flex items-center gap-1.5 text-xs text-muted-foreground'>
                <Building2Icon className='h-3 w-3' />
                <span className='truncate max-w-[120px]'>{request.organizationName}</span>
              </div>
            )}
          </div>
        )
      }
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => {
        const statusConfig = getBookingStatusBadgeStyle(row.original.status)

        return (
          <Badge variant='outline' className={cn('capitalize', statusConfig.className)}>
            {statusConfig.label}
          </Badge>
        )
      },
      filterFn: (row, _columnId, filterValue: BookingRequestStatus[]) => {
        if (!filterValue || filterValue.length === 0) return true
        return filterValue.includes(row.original.status)
      }
    },
    {
      accessorKey: 'createdAt',
      header: 'Submitted',
      cell: ({ row }) => {
        const { date } = formatDateTime(row.original.createdAt)
        return <span className='text-sm text-muted-foreground'>{date}</span>
      },
      sortingFn: (rowA, rowB) => {
        return new Date(rowA.original.createdAt).getTime() - new Date(rowB.original.createdAt).getTime()
      }
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }) => {
        const request = row.original
        const isPending = request.status === BookingRequestStatus.PENDING

        return (
          <div className='flex items-center justify-end gap-1'>
            {isPending && (
              <>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          variant='ghost'
                          size='icon'
                          className='h-8 w-8 text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50'
                          onClick={(e) => {
                            e.stopPropagation()
                            onApprove(request)
                          }}
                        />
                      }
                    >
                      <CheckIcon className='h-4 w-4' />
                    </TooltipTrigger>
                    <TooltipContent>Approve</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          variant='ghost'
                          size='icon'
                          className='h-8 w-8 text-red-600 hover:text-red-700 hover:bg-red-50'
                          onClick={(e) => {
                            e.stopPropagation()
                            onReject(request)
                          }}
                        />
                      }
                    >
                      <XIcon className='h-4 w-4' />
                    </TooltipTrigger>
                    <TooltipContent>Reject</TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              </>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant='ghost' size='icon' className='h-8 w-8' />}>
                <MoreHorizontalIcon className='h-4 w-4' />
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                <DropdownMenuItem onClick={() => onViewDetails(request)}>View Details</DropdownMenuItem>
                {isPending && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => onApprove(request)} className='text-emerald-600'>
                      <CheckIcon className='mr-2 h-4 w-4' />
                      Approve
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => onReject(request)} className='text-red-600'>
                      <XIcon className='mr-2 h-4 w-4' />
                      Reject
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )
      },
      size: 120
    }
  ]
}
