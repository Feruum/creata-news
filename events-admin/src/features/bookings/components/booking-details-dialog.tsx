import { type BookingRequest, BookingRequestStatus, PlatformRole } from '@studyverse/proto'
import { format, parseISO } from 'date-fns'
import { Building2Icon, CalendarIcon, ClockIcon, FileTextIcon, MapPinIcon, UserIcon, XIcon } from 'lucide-react'
import { useState } from 'react'
import { useOrganizationDetails, useOrganizationTypeDetails, useUserDetails } from '@/features/bookings'
import { getBookingPriorityBadgeStyle, getBookingStatusBadgeStyle } from '@/features/bookings/booking-badge-styles'
import { OrganizationInfoModal } from '@/features/organizations'
import { UserProfileModal } from '@/features/users'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { Separator } from '@/shared/ui/components/separator'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { cn } from '@/shared/ui/lib/utils'

function getRoleBadgeClass(role: PlatformRole): string {
  switch (role) {
    case PlatformRole.ADMIN:
      return 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400'
    case PlatformRole.STAFF:
      return 'bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-400'
    case PlatformRole.USER:
      return 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
  }
}

function getRoleLabel(role: PlatformRole): string {
  switch (role) {
    case PlatformRole.ADMIN:
      return 'Admin'
    case PlatformRole.STAFF:
      return 'Staff'
    case PlatformRole.USER:
      return 'User'
    default:
      return 'Unknown'
  }
}

interface RequesterInfoCardProps {
  requesterId: number
  requesterName?: string
}

function RequesterInfoCard({ requesterId, requesterName }: RequesterInfoCardProps) {
  const [imageError, setImageError] = useState(false)
  const { data, isLoading } = useUserDetails({ id: requesterId })
  const user = data?.user

  const avatarUrl =
    user?.avatarUrl || (user?.telegramUsername ? `https://t.me/i/userpic/320/${user.telegramUsername}.jpg` : null)

  const displayName =
    user?.firstName || user?.lastName
      ? `${user.firstName || ''} ${user.lastName || ''}`.trim()
      : requesterName || 'Unknown'

  const initials = displayName.slice(0, 2).toUpperCase()

  const handleImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget
    if (img.naturalWidth === 1 && img.naturalHeight === 1) {
      setImageError(true)
    }
  }

  if (isLoading) {
    return (
      <div className='flex items-center gap-3'>
        <Skeleton className='h-10 w-10 rounded-full' />
        <div className='space-y-1.5'>
          <Skeleton className='h-4 w-24' />
          <Skeleton className='h-5 w-12' />
        </div>
      </div>
    )
  }

  return (
    <div className='flex items-center gap-3'>
      {avatarUrl && !imageError ? (
        <img
          src={avatarUrl}
          alt={displayName}
          className='h-10 w-10 shrink-0 rounded-full object-cover'
          onError={() => setImageError(true)}
          onLoad={handleImageLoad}
        />
      ) : (
        <div className='flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-muted text-sm font-medium'>
          {initials}
        </div>
      )}
      <div>
        <div className='flex items-center gap-1.5'>
          <UserProfileModal userId={requesterId}>
            <button type='button' className='text-sm font-medium text-primary hover:underline cursor-pointer text-left'>
              {displayName}
            </button>
          </UserProfileModal>
          <AchievementNameBadge badge={user?.selectedBadge} userId={requesterId} />
        </div>
        {user && (
          <Badge variant='secondary' className={cn('text-xs mt-1', getRoleBadgeClass(user.platformRole))}>
            {getRoleLabel(user.platformRole)}
          </Badge>
        )}
      </div>
    </div>
  )
}

interface OrganizationInfoCardProps {
  organizationId: number
  organizationName?: string
}

function OrganizationInfoCard({ organizationId, organizationName }: OrganizationInfoCardProps) {
  const [imageError, setImageError] = useState(false)
  const { data: orgData, isLoading: orgLoading } = useOrganizationDetails({ id: organizationId })
  const org = orgData?.organization

  const { data: typeData } = useOrganizationTypeDetails({
    id: org?.organizationTypeId ?? 0,
    enabled: !!org?.organizationTypeId
  })
  const orgType = typeData?.organizationType

  const displayName = organizationName || org?.title || 'Unknown'
  const initials = displayName.slice(0, 2).toUpperCase()

  if (orgLoading) {
    return (
      <div className='flex items-center gap-3'>
        <Skeleton className='h-10 w-10 rounded-lg' />
        <div className='space-y-1.5'>
          <Skeleton className='h-4 w-32' />
          <Skeleton className='h-5 w-16' />
        </div>
      </div>
    )
  }

  return (
    <div className='flex items-center gap-3'>
      {org?.imageUrl && !imageError ? (
        <img
          src={org.imageUrl}
          alt={displayName}
          className='h-10 w-10 shrink-0 rounded-lg object-cover'
          onError={() => setImageError(true)}
        />
      ) : (
        <div className='flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted text-sm font-medium'>
          {initials}
        </div>
      )}
      <div>
        <OrganizationInfoModal organizationId={organizationId}>
          <button type='button' className='text-sm font-medium text-primary hover:underline cursor-pointer text-left'>
            {displayName}
          </button>
        </OrganizationInfoModal>
        {orgType && (
          <Badge
            variant='outline'
            className='text-xs mt-1 bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
          >
            {orgType.title}
          </Badge>
        )}
      </div>
    </div>
  )
}

function formatDateTime(isoString: string): string {
  try {
    return format(parseISO(isoString), 'MMM d, yyyy HH:mm')
  } catch {
    return 'Invalid date'
  }
}

interface BookingDetailsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: BookingRequest | null
  onApprove?: () => void
  onReject?: () => void
}

export function BookingDetailsDialog({ open, onOpenChange, request, onApprove, onReject }: BookingDetailsDialogProps) {
  if (!request) return null

  const statusConfig = getBookingStatusBadgeStyle(request.status)
  const priorityConfig = getBookingPriorityBadgeStyle(request.priority)
  const isPending = request.status === BookingRequestStatus.PENDING

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle className='flex items-center justify-between'>
            <span>Booking Request Details</span>
            <div className='flex gap-2'>
              <Badge variant='outline' className={cn(statusConfig.className)}>
                {statusConfig.label}
              </Badge>
              <Badge variant='outline' className={cn(priorityConfig.className)}>
                {priorityConfig.label}
              </Badge>
            </div>
          </DialogTitle>
        </DialogHeader>

        <div className='space-y-6 py-4'>
          <div>
            <h3 className='font-semibold text-lg'>{request.title}</h3>
            {request.description && <p className='text-muted-foreground mt-1'>{request.description}</p>}
          </div>

          <Separator />

          <div className='grid gap-4'>
            <div className='flex items-start gap-3'>
              <MapPinIcon className='h-5 w-5 text-muted-foreground mt-0.5' />
              <div>
                <p className='font-medium'>Room</p>
                <p className='text-sm text-muted-foreground'>
                  {request.room ? (
                    <>
                      {request.room.name}
                      <br />
                      {request.room.block} • Room {request.room.roomNumber} • Floor {request.room.floor}
                    </>
                  ) : (
                    'Unknown'
                  )}
                </p>
              </div>
            </div>

            <div className='flex items-start gap-3'>
              <CalendarIcon className='h-5 w-5 text-muted-foreground mt-0.5' />
              <div>
                <p className='font-medium'>Date & Time</p>
                <p className='text-sm text-muted-foreground'>
                  {formatDateTime(request.startTime)} - {format(parseISO(request.endTime), 'HH:mm')}
                </p>
              </div>
            </div>

            <div className='flex items-start gap-3'>
              <UserIcon className='h-5 w-5 text-muted-foreground mt-0.5' />
              <div>
                <p className='font-medium mb-2'>Requester</p>
                <RequesterInfoCard requesterId={request.requesterId} requesterName={request.requesterName} />
              </div>
            </div>

            {request.organizationId && (
              <div className='flex items-start gap-3'>
                <Building2Icon className='h-5 w-5 text-muted-foreground mt-0.5' />
                <div>
                  <p className='font-medium mb-2'>Organization</p>
                  <OrganizationInfoCard
                    organizationId={request.organizationId}
                    organizationName={request.organizationName}
                  />
                </div>
              </div>
            )}

            <div className='flex items-start gap-3'>
              <ClockIcon className='h-5 w-5 text-muted-foreground mt-0.5' />
              <div>
                <p className='font-medium'>Submitted</p>
                <p className='text-sm text-muted-foreground'>{formatDateTime(request.createdAt)}</p>
              </div>
            </div>

            {request.status === BookingRequestStatus.REJECTED && request.rejectionReason && (
              <div className='flex items-start gap-3'>
                <XIcon className='h-5 w-5 text-red-500 mt-0.5' />
                <div>
                  <p className='font-medium text-red-600'>Rejection Reason</p>
                  <p className='text-sm text-muted-foreground'>{request.rejectionReason}</p>
                </div>
              </div>
            )}

            {request.reviewedAt && (
              <div className='flex items-start gap-3'>
                <FileTextIcon className='h-5 w-5 text-muted-foreground mt-0.5' />
                <div>
                  <p className='font-medium'>Reviewed</p>
                  <p className='text-sm text-muted-foreground'>{formatDateTime(request.reviewedAt)}</p>
                </div>
              </div>
            )}
          </div>

          {isPending && onApprove && onReject && (
            <>
              <Separator />
              <div className='flex justify-end gap-2'>
                <Button variant='outline' onClick={onReject}>
                  <XIcon className='mr-2 h-4 w-4' />
                  Reject
                </Button>
                <Button onClick={onApprove} className='bg-emerald-600 hover:bg-emerald-700'>
                  Approve
                </Button>
              </div>
            </>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
