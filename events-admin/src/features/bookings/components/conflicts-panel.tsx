import type { BookingConflict, BookingRequest } from '@studyverse/proto'
import { format, parseISO } from 'date-fns'
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ClockIcon,
  CrownIcon,
  MapPinIcon,
  UsersIcon
} from 'lucide-react'
import { useMemo, useState } from 'react'

import { useBookingConflicts } from '@/features/bookings'
import { getBookingPriorityBadgeStyle } from '@/features/bookings/booking-badge-styles'

import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/shared/ui/components/collapsible'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { cn } from '@/shared/ui/lib/utils'

function formatTime(isoString: string): string {
  try {
    return format(parseISO(isoString), 'HH:mm')
  } catch {
    return 'Invalid'
  }
}

function formatDate(isoString: string): string {
  try {
    return format(parseISO(isoString), 'MMM d, yyyy')
  } catch {
    return 'Invalid'
  }
}

interface ConflictsPanelProps {
  onSelectRequest?: (request: BookingRequest) => void
  onApproveRequest?: (request: BookingRequest) => void
}

export function ConflictsPanel({ onSelectRequest, onApproveRequest }: ConflictsPanelProps) {
  const dateRange = useMemo(() => {
    const now = new Date()
    const thirtyDaysLater = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000)
    return {
      fromDate: now.toISOString(),
      toDate: thirtyDaysLater.toISOString()
    }
  }, [])

  const { data, isLoading, error } = useBookingConflicts(dateRange)

  if (isLoading) {
    return (
      <Card>
        <CardHeader className='pb-3'>
          <Skeleton className='h-6 w-48' />
          <Skeleton className='h-4 w-64' />
        </CardHeader>
        <CardContent className='space-y-3'>
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-24 w-full' />
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className='border-red-200 bg-red-50'>
        <CardHeader className='pb-3'>
          <CardTitle className='flex items-center gap-2 text-red-700'>
            <AlertTriangleIcon className='h-5 w-5' />
            Failed to Load Conflicts
          </CardTitle>
          <CardDescription className='text-red-600'>{error.message}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const conflicts = data?.conflicts ?? []

  if (conflicts.length === 0) {
    return null
  }

  return (
    <Card className='border-amber-200 bg-amber-50/50'>
      <CardHeader className='pb-3'>
        <CardTitle className='flex items-center gap-2 text-amber-700'>
          <AlertTriangleIcon className='h-5 w-5' />
          Scheduling Conflicts
          <Badge
            variant='outline'
            className='ml-2 border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300'
          >
            {conflicts.length}
          </Badge>
        </CardTitle>
        <CardDescription>
          Multiple pending requests for the same room and time. Suggested winner is based on priority.
        </CardDescription>
      </CardHeader>
      <CardContent className='space-y-3'>
        {conflicts.map((conflict: BookingConflict, index: number) => (
          <ConflictItem
            key={`${conflict.roomId}-${conflict.startTime}-${index}`}
            conflict={conflict}
            onSelectRequest={onSelectRequest}
            onApproveRequest={onApproveRequest}
          />
        ))}
      </CardContent>
    </Card>
  )
}

interface ConflictItemProps {
  conflict: BookingConflict
  onSelectRequest?: (request: BookingRequest) => void
  onApproveRequest?: (request: BookingRequest) => void
}

function ConflictItem({ conflict, onSelectRequest, onApproveRequest }: ConflictItemProps) {
  const [isOpen, setIsOpen] = useState(false)

  const roomName = conflict.conflictingRequests[0]?.room?.name ?? 'Unknown Room'
  const suggestedWinner = conflict.suggestedWinner
  const otherRequests = conflict.conflictingRequests.filter((r) => r.id !== suggestedWinner?.id)

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <div className='rounded-lg border border-amber-200 bg-white overflow-hidden'>
        <CollapsibleTrigger
          render={
            <button
              type='button'
              className='flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-amber-50/50 transition-colors'
            />
          }
        >
          {isOpen ? (
            <ChevronDownIcon className='h-4 w-4 text-muted-foreground shrink-0' />
          ) : (
            <ChevronRightIcon className='h-4 w-4 text-muted-foreground shrink-0' />
          )}
          <div className='flex-1 min-w-0'>
            <div className='flex items-center gap-2'>
              <MapPinIcon className='h-4 w-4 text-muted-foreground' />
              <span className='font-medium truncate'>{roomName}</span>
              <Badge variant='outline' className='text-xs'>
                {conflict.conflictingRequests.length} requests
              </Badge>
            </div>
            <div className='flex items-center gap-2 mt-1 text-sm text-muted-foreground'>
              <ClockIcon className='h-3.5 w-3.5' />
              <span>
                {formatDate(conflict.startTime)} • {formatTime(conflict.startTime)} - {formatTime(conflict.endTime)}
              </span>
            </div>
          </div>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className='border-t border-amber-100 px-4 py-3 space-y-3'>
            {suggestedWinner && (
              <div className='rounded-lg border-2 border-emerald-300 bg-emerald-50 p-3'>
                <div className='flex items-center gap-2 mb-2'>
                  <CrownIcon className='h-4 w-4 text-emerald-600' />
                  <span className='text-xs font-medium text-emerald-700 uppercase tracking-wide'>
                    Suggested Winner (Highest Priority)
                  </span>
                </div>
                <RequestCard
                  request={suggestedWinner}
                  isHighlighted
                  onSelect={onSelectRequest}
                  onApprove={onApproveRequest}
                />
              </div>
            )}

            {otherRequests.length > 0 && (
              <div className='space-y-2'>
                <p className='text-xs font-medium text-muted-foreground uppercase tracking-wide'>
                  Conflicting Requests
                </p>
                {otherRequests.map((request) => (
                  <RequestCard key={request.id} request={request} onSelect={onSelectRequest} />
                ))}
              </div>
            )}
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  )
}

interface RequestCardProps {
  request: BookingRequest
  isHighlighted?: boolean
  onSelect?: (request: BookingRequest) => void
  onApprove?: (request: BookingRequest) => void
}

function RequestCard({ request, isHighlighted, onSelect, onApprove }: RequestCardProps) {
  const priorityConfig = getBookingPriorityBadgeStyle(request.priority)

  return (
    <div
      className={cn(
        'rounded-md border p-3',
        isHighlighted ? 'border-emerald-200 bg-white' : 'border-gray-200 bg-gray-50'
      )}
    >
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0 flex-1'>
          <div className='flex items-center gap-2'>
            <button
              type='button'
              onClick={() => onSelect?.(request)}
              className='font-medium text-sm hover:underline truncate text-left'
            >
              {request.title}
            </button>
            <Badge variant='outline' className={cn('text-xs shrink-0', priorityConfig.className)}>
              {priorityConfig.label}
            </Badge>
          </div>
          <div className='flex items-center gap-3 mt-1 text-xs text-muted-foreground'>
            <span className='flex items-center gap-1'>
              <UsersIcon className='h-3 w-3' />
              {request.requesterName || 'Unknown'}
            </span>
            {request.organizationName && <span className='truncate'>{request.organizationName}</span>}
          </div>
        </div>
        {isHighlighted && onApprove && (
          <Button size='sm' onClick={() => onApprove(request)} className='bg-emerald-600 hover:bg-emerald-700 shrink-0'>
            Approve
          </Button>
        )}
      </div>
    </div>
  )
}
