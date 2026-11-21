import { Code, ConnectError, createClient } from '@connectrpc/connect'
import { useTransport } from '@connectrpc/connect-query'
import { CheckInPhase, EventFormat, EventStatus, LocationType, RoomsService } from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'
import { format } from 'date-fns'
import {
  CalendarIcon,
  CheckCircle2Icon,
  CopyIcon,
  GlobeIcon,
  ImageIcon,
  Loader2,
  MapPinIcon,
  PencilIcon,
  SendIcon,
  Trash2Icon,
  XCircleIcon,
  XIcon
} from 'lucide-react'
import type { Value } from 'platejs'
import { Suspense, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import { bookingConflictsKey, bookingRequestsListKey } from '@/features/bookings'
import {
  useCancelEvent,
  useDeleteEvent,
  useEvent,
  useEventPublishReadiness,
  usePublishEvent,
  useReviewEventBadges,
  useUpdateEvent
} from '@/features/events'
import { RegistrationWorkspaceLink } from '@/features/registration-forms'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { EMPTY_PLATE_VALUE, parsePlateJson } from '@/shared/lib/plate-utils'
import { getRequestErrorMessage } from '@/shared/lib/request-errors'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/shared/ui/components/alert-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import { Badge } from '@/shared/ui/components/badge'
import { Button, buttonVariants } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/shared/ui/components/dialog'
import { cn } from '@/shared/ui/lib/utils'
import { eventPublishReadinessKey } from '../api/keys'
import { isBadgeRequested } from '../utils/event-badges'
import { formatEventDate, formatEventTime, getRelativeTime } from '../utils/event-date-helpers'
import { canRequestBooking, normalizeEventRoomId, readinessMessage } from '../utils/event-submission'
import { copyPublicEventLink } from '../utils/event-workflows'
import { formatStructuredLocation } from '../utils/format-location'
import { CreateEventForm } from './create-event-form'
import { DescriptionEditorDialog } from './description-editor-dialog'
import { EventBadgeReview } from './event-badge-review'
import { EventBadges } from './event-badges'
import { EventCheckInPanel } from './event-check-in-panel'
import { EventDetailsSkeleton } from './event-details-skeleton'
import { EventRegistrationsSection } from './event-registrations-section'
import { PlateEditor } from './plate-editor'

// ============================================================================
// EVENT DETAILS CONTENT
// ============================================================================

interface EventDetailsContentProps {
  eventId: number
  onClose: () => void
}

function getEventStatusLabel(status: EventStatus): string {
  switch (status) {
    case EventStatus.DRAFT:
      return 'Draft'
    case EventStatus.PUBLISHED:
      return 'Published'
    case EventStatus.ARCHIVED:
      return 'Archived'
    case EventStatus.CANCELLED:
      return 'Cancelled'
    default:
      return 'Unknown'
  }
}

function getCheckInPhaseLabel(phase: CheckInPhase): string {
  switch (phase) {
    case CheckInPhase.NONE:
      return 'Not started'
    case CheckInPhase.LIVE:
      return 'Live'
    case CheckInPhase.GRACE:
      return 'Grace period'
    case CheckInPhase.LOCKED:
      return 'Locked'
    default:
      return 'Unknown'
  }
}

function EventDetailsContent({ eventId, onClose }: EventDetailsContentProps) {
  const [isEditFormOpen, setIsEditFormOpen] = useState(false)
  const [isDescriptionEditorOpen, setIsDescriptionEditorOpen] = useState(false)
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [isCheckingPublishReadiness, setIsCheckingPublishReadiness] = useState(false)
  const [isRequestingRoomBooking, setIsRequestingRoomBooking] = useState(false)
  const [workflowError, setWorkflowError] = useState<string | null>(null)
  const publishLock = useRef(false)
  const transport = useTransport()
  const queryClient = useQueryClient()
  const roomsClient = useMemo(() => createClient(RoomsService, transport), [transport])
  const [descriptionDraft, setDescriptionDraft] = useState<Value>(EMPTY_PLATE_VALUE)

  const { data } = useEvent(eventId)
  const event = data?.event
  const canEdit = data?.canEdit ?? false
  const canReviewBadges = data?.canReviewBadges ?? false
  const canDelete = data?.canDelete ?? false
  const canManageRegistrations = data?.canManageRegistrations ?? false
  const publishReadiness = useEventPublishReadiness(eventId, event?.status === EventStatus.DRAFT && canEdit)

  const descriptionMutation = useUpdateEvent(false)
  const badgeReviewMutation = useReviewEventBadges()
  const deleteMutation = useDeleteEvent()
  const publishMutation = usePublishEvent(false)
  const cancelMutation = useCancelEvent()
  const isPublishPending =
    isCheckingPublishReadiness || isRequestingRoomBooking || publishReadiness.isFetching || publishMutation.isPending
  const canPublishNow = !publishReadiness.isError && publishReadiness.data?.canPublishNow === true
  const roomId = event?.structuredLocation?.roomId
  const isCampusRoom =
    event?.structuredLocation?.type === LocationType.CAMPUS_ROOM ||
    event?.structuredLocation?.type === LocationType.HYBRID
  const canRequestRoomBooking =
    canEdit &&
    !publishReadiness.isError &&
    canRequestBooking(publishReadiness.data?.state) &&
    isCampusRoom &&
    Boolean(roomId?.trim())

  const handleCopyLink = async () => {
    if (!event) return
    try {
      const mapOrigin =
        import.meta.env.VITE_MAP_URL || (import.meta.env.DEV ? 'http://localhost:3001' : window.location.origin)
      await copyPublicEventLink(event, mapOrigin, navigator.clipboard)
      toast.success('Link copied')
    } catch {
      toast.error('Could not copy link. Please try again.')
    }
  }

  const handleOpenDescriptionEditor = () => {
    setDescriptionDraft(parsePlateJson(event?.description))
    setIsDescriptionEditorOpen(true)
  }

  const handleSaveDescription = async (value: Value) => {
    if (!event) return

    try {
      await descriptionMutation.mutateAsync({
        id: event.id,
        title: event.title,
        description: JSON.stringify(value),
        structuredLocation: event.structuredLocation,
        startTime: event.startTime,
        endTime: event.endTime || event.startTime
      })
      toast.success('Description updated')
      setIsDescriptionEditorOpen(false)
    } catch (error) {
      toast.error(getRequestErrorMessage(error))
    }
  }

  const handleDelete = async () => {
    if (!event) return
    try {
      await deleteMutation.mutateAsync({ id: event.id })
      toast.success('Event deleted')
      setIsDeleteDialogOpen(false)
      onClose()
    } catch {
      /* The mutation cache displays this failure. */
    }
  }

  const handlePublish = async () => {
    if (!event || !canEdit || isPublishPending || publishLock.current) return
    publishLock.current = true
    setWorkflowError(null)
    setIsCheckingPublishReadiness(true)
    try {
      const readiness = await publishReadiness.refetch()
      if (readiness.error || !readiness.data?.canPublishNow) return
      await publishMutation.mutateAsync({ eventId: event.id })
      toast.success('Event published')
    } catch (error) {
      setWorkflowError(getRequestErrorMessage(error, 'publish'))
      await publishReadiness.refetch()
    } finally {
      publishLock.current = false
      setIsCheckingPublishReadiness(false)
    }
  }

  const handleRequestRoomBooking = async () => {
    if (!event || !roomId?.trim() || !canRequestRoomBooking || isPublishPending || publishLock.current) return
    publishLock.current = true
    setWorkflowError(null)
    setIsRequestingRoomBooking(true)
    try {
      const readiness = await publishReadiness.refetch()
      if (readiness.error || !canRequestBooking(readiness.data?.state)) return
      await roomsClient.createBookingRequest({
        roomId: normalizeEventRoomId(roomId),
        eventId: event.id,
        organizationId: event.organizationId,
        title: event.title,
        startTime: event.startTime,
        endTime: event.endTime
      })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: bookingRequestsListKey() }),
        queryClient.invalidateQueries({ queryKey: bookingConflictsKey() }),
        queryClient.invalidateQueries({
          queryKey: eventPublishReadinessKey({ eventId: event.id }),
          refetchType: 'none'
        })
      ])
      await publishReadiness.refetch()
    } catch (error) {
      await publishReadiness.refetch()
      if (!(error instanceof ConnectError && error.code === Code.AlreadyExists)) {
        setWorkflowError(getRequestErrorMessage(error, 'booking'))
      }
    } finally {
      publishLock.current = false
      setIsRequestingRoomBooking(false)
    }
  }

  const handleRefreshReadiness = () => {
    setWorkflowError(null)
    return publishReadiness.refetch()
  }

  const handleCancelEvent = async () => {
    if (!event) return
    try {
      await cancelMutation.mutateAsync({ eventId: event.id })
      toast.success('Event cancelled')
    } catch {
      /* The mutation cache displays this failure. */
    }
  }

  if (!event) {
    return (
      <div className='flex h-64 items-center justify-center'>
        <p className='text-sm text-muted-foreground'>Event not found</p>
      </div>
    )
  }

  const timeDisplay = formatEventTime(event.startTime, event.endTime)
  const relativeTime = getRelativeTime(event.startTime)
  const org = event.organization

  // Prepare initial values for edit form
  const startDate = new Date(event.startTime)
  const endDate = event.endTime ? new Date(event.endTime) : new Date(event.startTime)
  const isAllDay =
    startDate.getHours() === 0 &&
    startDate.getMinutes() === 0 &&
    endDate.getHours() === 23 &&
    endDate.getMinutes() === 59

  const editFormInitialValues = {
    title: event.title,
    socGpaRequested: isBadgeRequested(event.socGpaBadge),
    irosRequested: isBadgeRequested(event.irosBadge),
    description: event.description || '',
    hasTime: !isAllDay,
    startDate,
    endDate,
    startTime: isAllDay ? '' : format(startDate, 'HH:mm'),
    endTime: isAllDay ? '' : format(endDate, 'HH:mm'),
    organizationIds: event.organization ? [event.organization.id] : [],
    categoryId: event.category?.id,
    locationType: event.structuredLocation?.type ?? LocationType.CAMPUS_ROOM,
    roomId: event.structuredLocation?.roomId ?? '',
    roomName: event.structuredLocation?.roomName ?? '',
    floor: event.structuredLocation?.floor,
    externalAddress: event.structuredLocation?.externalAddress ?? '',
    googleMapsUrl: event.structuredLocation?.googleMapsUrl ?? '',
    onlineUrl: event.structuredLocation?.onlineUrl ?? '',
    onlinePlatform: event.structuredLocation?.onlinePlatform ?? ''
  }

  return (
    <>
      {/* Header */}
      <div className='flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4'>
        <div className='min-w-0 flex-1'>
          <span className='text-xs text-muted-foreground'>{relativeTime}</span>
          <DialogTitle className='text-lg font-semibold leading-tight'>{event.title}</DialogTitle>
          <DialogDescription className='sr-only'>Event details for {event.title}</DialogDescription>
          <EventBadges event={event} showRequests className='mt-2' />
        </div>

        <div className='flex items-center gap-1 shrink-0'>
          <RegistrationWorkspaceLink
            eventId={event.id}
            canEdit={canEdit}
            canManageRegistrations={canManageRegistrations}
          />
          {event.status === EventStatus.DRAFT && (
            <Button
              variant='ghost'
              size='icon'
              className='h-8 w-8 text-green-600 hover:text-green-700 hover:bg-green-50'
              onClick={handlePublish}
              disabled={!canEdit || !canPublishNow || isPublishPending}
              aria-label='Publish event'
            >
              {isPublishPending ? (
                <Loader2 className='h-4 w-4 animate-spin' aria-hidden='true' />
              ) : (
                <SendIcon className='h-4 w-4' aria-hidden='true' />
              )}
            </Button>
          )}
          {event.status === EventStatus.PUBLISHED && (
            <Button
              variant='ghost'
              size='icon'
              className='h-8 w-8 text-amber-600 hover:text-amber-700 hover:bg-amber-50'
              onClick={handleCancelEvent}
              disabled={!canEdit || cancelMutation.isPending}
              aria-label='Cancel event'
            >
              {cancelMutation.isPending ? (
                <Loader2 className='h-4 w-4 animate-spin' aria-hidden='true' />
              ) : (
                <XCircleIcon className='h-4 w-4' aria-hidden='true' />
              )}
            </Button>
          )}
          <Button variant='ghost' size='icon' className='h-8 w-8' onClick={handleCopyLink} aria-label='Copy link'>
            <CopyIcon className='h-4 w-4' aria-hidden='true' />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            className='h-8 w-8'
            onClick={() => setIsEditFormOpen(true)}
            disabled={!canEdit}
            aria-label='Edit event'
          >
            <PencilIcon className='h-4 w-4' aria-hidden='true' />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            className='h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10'
            onClick={() => setIsDeleteDialogOpen(true)}
            disabled={!canDelete}
            aria-label='Delete event'
          >
            <Trash2Icon className='h-4 w-4' aria-hidden='true' />
          </Button>
          <Button variant='ghost' size='icon' aria-label='Close' className='h-8 w-8' onClick={onClose}>
            <XIcon className='h-4 w-4' aria-hidden='true' />
          </Button>
        </div>
      </div>

      {/* Content - scrollable area */}
      <div className='flex-1 overflow-y-auto'>
        <div className='p-6'>
          {(canEdit || canReviewBadges) && (
            <div className='mb-6'>
              <EventBadgeReview
                event={event}
                canReview={canReviewBadges}
                isPending={badgeReviewMutation.isPending}
                hasError={badgeReviewMutation.isError}
                onReview={(badge, status) => {
                  badgeReviewMutation.mutate({
                    eventId: event.id,
                    ...(badge === 'socGpa' ? { socGpaStatus: status } : { irosStatus: status })
                  })
                }}
              />
            </div>
          )}
          {/* Top section: Image + Key Info side by side */}
          <div className='flex gap-6 mb-6'>
            {/* Square Cover Image */}
            <div className='shrink-0'>
              <div className='relative w-48 h-48 overflow-hidden rounded-xl border bg-muted/30 shadow-sm'>
                {event.imageUrl ? (
                  <img src={event.imageUrl} alt={`Cover for ${event.title}`} className='h-full w-full object-cover' />
                ) : (
                  <div className='flex h-full w-full flex-col items-center justify-center gap-2 text-muted-foreground'>
                    <ImageIcon className='h-10 w-10 opacity-40' aria-hidden='true' />
                    <span className='text-xs'>No cover</span>
                  </div>
                )}
              </div>
            </div>

            {/* Key Info beside image */}
            <div className='flex-1 min-w-0 space-y-4'>
              {/* Date & Time Card */}
              <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
                <div className='flex items-start gap-4'>
                  <div className='flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary'>
                    <CalendarIcon className='h-6 w-6' aria-hidden='true' />
                  </div>
                  <div className='flex-1 min-w-0'>
                    <p className='text-sm font-semibold'>{formatEventDate(event.startTime, event.endTime)}</p>
                    <p className='text-sm text-muted-foreground'>{timeDisplay || 'All day event'}</p>
                  </div>
                </div>
              </div>

              {/* Format & Location Card */}
              <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
                <div className='flex items-start gap-4'>
                  <div
                    className={cn(
                      'flex h-12 w-12 items-center justify-center rounded-lg',
                      event.format === EventFormat.ONLINE
                        ? 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950 dark:text-emerald-400'
                        : event.format === EventFormat.HYBRID
                          ? 'bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400'
                          : 'bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400'
                    )}
                  >
                    {event.format === EventFormat.ONLINE ? (
                      <GlobeIcon className='h-6 w-6' aria-hidden='true' />
                    ) : event.format === EventFormat.HYBRID ? (
                      <>
                        <GlobeIcon className='h-5 w-5 absolute -translate-x-1' aria-hidden='true' />
                        <MapPinIcon className='h-5 w-5 absolute translate-x-1' aria-hidden='true' />
                      </>
                    ) : (
                      <MapPinIcon className='h-6 w-6' aria-hidden='true' />
                    )}
                  </div>
                  <div className='flex-1 min-w-0'>
                    <div className='flex items-center gap-2'>
                      <p className='text-sm font-semibold'>
                        {event.format === EventFormat.ONLINE
                          ? 'Online Event'
                          : event.format === EventFormat.HYBRID
                            ? 'Hybrid Event'
                            : 'In-Person Event'}
                      </p>
                      <Badge
                        variant='secondary'
                        className={cn(
                          'text-xs',
                          event.format === EventFormat.ONLINE
                            ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-400'
                            : event.format === EventFormat.HYBRID
                              ? 'bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-400'
                              : 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400'
                        )}
                      >
                        {event.format === EventFormat.ONLINE
                          ? 'Virtual'
                          : event.format === EventFormat.HYBRID
                            ? 'Virtual + Physical'
                            : 'Physical'}
                      </Badge>
                    </div>
                    {event.structuredLocation && (
                      <p className='text-sm text-muted-foreground truncate'>
                        {formatStructuredLocation(event.structuredLocation)}
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Bottom section: Organization + Category + Description */}
          <div className='grid gap-4 md:grid-cols-[200px_1fr]'>
            {/* Left: Org & Category */}
            <div className='space-y-4'>
              {/* Organization */}
              {org && (
                <div className='rounded-lg border bg-muted/30 p-4'>
                  <p className='text-xs font-medium uppercase tracking-wide text-muted-foreground mb-3'>Organized by</p>
                  <div className='flex items-center gap-3'>
                    <Avatar className='h-10 w-10 border shadow-sm'>
                      <AvatarImage src={org.imageUrl ?? undefined} alt={`${org.title} logo`} />
                      <AvatarFallback className='text-sm font-medium'>
                        {org.title?.[0]?.toUpperCase() || 'O'}
                      </AvatarFallback>
                    </Avatar>
                    <span className='text-sm font-medium truncate'>{org.title}</span>
                  </div>
                </div>
              )}

              {/* Category */}
              {event.category && (
                <div className='rounded-lg border bg-muted/30 p-4'>
                  <p className='text-xs font-medium uppercase tracking-wide text-muted-foreground mb-3'>Category</p>
                  <div className='flex items-center gap-3'>
                    <div
                      className='h-4 w-4 rounded-full shrink-0 shadow-sm'
                      style={{ backgroundColor: event.category.color || '#6B7280' }}
                    />
                    <span className='text-sm font-medium'>{event.category.name}</span>
                  </div>
                </div>
              )}

              {/* Event ID */}
              <div className='text-center pt-2'>
                <span className='text-xs text-muted-foreground font-mono'>Event #{event.id}</span>
              </div>
            </div>

            {/* Right: Description */}
            <button
              type='button'
              onClick={handleOpenDescriptionEditor}
              disabled={!canEdit}
              className='h-fit rounded-lg border bg-muted/30 p-4 text-left transition-colors hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
            >
              <h3 className='mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground'>Description</h3>
              {event.description ? (
                <div className='line-clamp-[12] pointer-events-none'>
                  <PlateEditor value={parsePlateJson(event.description)} onChange={() => {}} readOnly />
                </div>
              ) : (
                <p className='text-sm text-muted-foreground italic'>Click to add a description...</p>
              )}
            </button>
          </div>

          {/* Event Metadata */}
          <div className='mt-4 rounded-lg border bg-muted/20 p-4'>
            <h3 className='mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground'>Event Metadata</h3>
            <div className='grid gap-2 text-xs sm:grid-cols-2 lg:grid-cols-3'>
              <div>
                <p className='text-muted-foreground'>Status</p>
                <p className='font-medium'>{getEventStatusLabel(event.status)}</p>
              </div>
              <div>
                <p className='text-muted-foreground'>Slug</p>
                <p className='font-mono break-all'>{event.slug || '—'}</p>
              </div>
              <div>
                <p className='text-muted-foreground'>Check-in Phase</p>
                <p className='font-medium'>{getCheckInPhaseLabel(event.checkInPhase)}</p>
              </div>
              <div>
                <p className='text-muted-foreground'>Allocated seats</p>
                <p className='font-medium'>
                  {event.allocatedSeatCount}
                  {event.capacity ? ` / ${event.capacity}` : ''}
                </p>
                <p className='text-xs text-muted-foreground'>
                  Map: {event.totalRegistrations} · Sunday Open: {event.externalRegistrationCount}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground'>Account Attendees</p>
                <p className='font-medium'>{event.totalAttendees}</p>
              </div>
              <div>
                <p className='text-muted-foreground'>Created</p>
                <p className='font-medium'>
                  {event.createdAt ? format(new Date(event.createdAt), 'MMM d, yyyy HH:mm') : '—'}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground'>Updated</p>
                <p className='font-medium'>
                  {event.updatedAt ? format(new Date(event.updatedAt), 'MMM d, yyyy HH:mm') : '—'}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground'>Published</p>
                <p className='font-medium'>
                  {event.publishedAt ? format(new Date(event.publishedAt), 'MMM d, yyyy HH:mm') : '—'}
                </p>
              </div>
              <div>
                <p className='text-muted-foreground'>Cancelled</p>
                <p className='font-medium'>
                  {event.cancelledAt ? format(new Date(event.cancelledAt), 'MMM d, yyyy HH:mm') : '—'}
                </p>
              </div>
            </div>
            {event.cancelReason && (
              <div className='mt-3 rounded-md border border-amber-200 bg-amber-50 p-2 text-xs text-amber-800 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300'>
                <p className='font-semibold'>Cancel reason</p>
                <p>{event.cancelReason}</p>
              </div>
            )}
          </div>

          {(event.organizers.length > 0 || event.collaborators.length > 0) && (
            <div className='mt-4 grid gap-4 lg:grid-cols-2'>
              {event.organizers.length > 0 && (
                <div className='rounded-lg border bg-muted/20 p-4'>
                  <h3 className='mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground'>Organizers</h3>
                  <div className='space-y-2'>
                    {event.organizers.map((organizer) => (
                      <div key={organizer.id} className='rounded-md border bg-background/60 p-2 text-xs'>
                        <p className='flex items-center gap-1.5 font-medium'>
                          {organizer.firstName || organizer.lastName
                            ? `${organizer.firstName || ''} ${organizer.lastName || ''}`.trim()
                            : organizer.username || `User #${organizer.userId}`}
                          <AchievementNameBadge badge={organizer.selectedBadge} userId={organizer.userId} />
                        </p>
                        <p className='text-muted-foreground'>{organizer.email || '—'}</p>
                        <p className='text-muted-foreground'>
                          Org: {organizer.organizationTitle || organizer.organizationId} • Role: {organizer.role || '—'}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {event.collaborators.length > 0 && (
                <div className='rounded-lg border bg-muted/20 p-4'>
                  <h3 className='mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground'>
                    Collaborators
                  </h3>
                  <div className='space-y-2'>
                    {event.collaborators.map((collaborator) => (
                      <div key={collaborator.id} className='rounded-md border bg-background/60 p-2 text-xs'>
                        <p className='font-medium'>
                          {collaborator.organizationTitle || `Org #${collaborator.organizationId}`}
                        </p>
                        <p className='text-muted-foreground'>Org ID: {collaborator.organizationId}</p>
                        <p className='text-muted-foreground'>
                          Added: {format(new Date(collaborator.addedAt), 'MMM d, yyyy HH:mm')}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Registrations Section */}
          {canManageRegistrations && (
            <div className='mt-6'>
              <EventRegistrationsSection
                eventId={event.id}
                isCheckInActive={event.checkInPhase === CheckInPhase.LIVE || event.checkInPhase === CheckInPhase.GRACE}
              />
            </div>
          )}

          <EventCheckInPanel
            eventId={event.id}
            canManageRegistrations={canManageRegistrations}
            eventStatus={event.status}
          />

          {/* Status Actions */}
          {event.status === EventStatus.DRAFT && (
            <div className='mt-4 rounded-lg border border-amber-200 bg-amber-50 dark:border-amber-900 dark:bg-amber-950/30 p-4'>
              <div className='flex items-center gap-3'>
                <CheckCircle2Icon className='h-5 w-5 text-amber-600 dark:text-amber-400 shrink-0' />
                <div className='flex-1'>
                  <p className='text-sm font-medium text-amber-800 dark:text-amber-300'>This event is a draft</p>
                  <p className='text-xs text-amber-600 dark:text-amber-400'>
                    Publish it to make it visible to students.
                  </p>
                </div>
                <Button
                  size='sm'
                  onClick={handlePublish}
                  disabled={!canEdit || !canPublishNow || isPublishPending}
                  className='bg-amber-600 hover:bg-amber-700 text-white shrink-0'
                >
                  {isPublishPending && <Loader2 className='mr-2 h-3 w-3 animate-spin' />}
                  Publish
                </Button>
              </div>
              {canEdit && (
                <div className='mt-3 space-y-2 text-sm' aria-live='polite'>
                  {workflowError && !publishReadiness.isError && (
                    <p role='alert' className='text-destructive'>
                      {workflowError}
                    </p>
                  )}
                  {publishReadiness.isError ? (
                    <div role='alert' className='space-y-2'>
                      <p>{getRequestErrorMessage(publishReadiness.error, 'booking')}</p>
                      <Button
                        variant='outline'
                        size='sm'
                        disabled={isPublishPending}
                        onClick={() => void handleRefreshReadiness()}
                      >
                        Retry room approval
                      </Button>
                    </div>
                  ) : publishReadiness.data ? (
                    <>
                      <p>{readinessMessage(publishReadiness.data, false)}</p>
                      {publishReadiness.data.rejectionReason && <p>{publishReadiness.data.rejectionReason}</p>}
                      <div className='flex flex-wrap gap-2'>
                        {canRequestRoomBooking && (
                          <Button size='sm' disabled={isPublishPending} onClick={handleRequestRoomBooking}>
                            {isRequestingRoomBooking && <Loader2 className='mr-2 h-3 w-3 animate-spin' />}
                            Request room booking
                          </Button>
                        )}
                        <Button
                          variant='outline'
                          size='sm'
                          disabled={isPublishPending}
                          onClick={() => void handleRefreshReadiness()}
                        >
                          Refresh room approval
                        </Button>
                      </div>
                    </>
                  ) : (
                    <p>Checking publish readiness...</p>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <AlertDialogContent style={{ overscrollBehavior: 'contain' }}>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Event</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the event "{event.title}".
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
              className={buttonVariants({ variant: 'destructive' })}
            >
              {deleteMutation.isPending && <Loader2 className='mr-2 h-4 w-4 animate-spin' aria-hidden='true' />}
              {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Description Editor Popup */}
      <DescriptionEditorDialog
        open={isDescriptionEditorOpen}
        onOpenChange={setIsDescriptionEditorOpen}
        initialValue={descriptionDraft}
        onSave={handleSaveDescription}
        isSaving={descriptionMutation.isPending}
        contentClassName='grid max-h-[90vh] w-[min(900px,calc(100vw-2rem))] grid-rows-[auto_1fr] gap-0 overflow-hidden p-0'
        placeholder='Write your event description...'
        onCancel={() => setDescriptionDraft(parsePlateJson(event?.description))}
      />

      {/* Edit Event Form */}
      {isEditFormOpen && (
        <CreateEventForm
          open={isEditFormOpen}
          onOpenChange={setIsEditFormOpen}
          eventId={event.id}
          initialValues={editFormInitialValues}
        />
      )}
    </>
  )
}

// ============================================================================
// MAIN MODAL COMPONENT (with trigger)
// ============================================================================

interface EventDetailsModalProps {
  eventId: number
  children: React.ReactNode
}

export function EventDetailsModal({ eventId, children }: EventDetailsModalProps) {
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <button type='button' onClick={() => setOpen(true)} className='w-full text-left'>
        {children}
      </button>
      <DialogContent
        className='flex max-h-[90vh] w-[min(900px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <Suspense fallback={<EventDetailsSkeleton />}>
          <EventDetailsContent eventId={eventId} onClose={() => setOpen(false)} />
        </Suspense>
      </DialogContent>
    </Dialog>
  )
}

// ============================================================================
// CONTROLLED VERSION (for external state management)
// ============================================================================

interface EventDetailsModalControlledProps {
  eventId: number | null
  onClose: () => void
}

export function EventDetailsModalControlled({ eventId, onClose }: EventDetailsModalControlledProps) {
  if (!eventId) return null

  return (
    <Dialog open={!!eventId} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className='flex max-h-[90vh] w-[min(900px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <Suspense fallback={<EventDetailsSkeleton />}>
          <EventDetailsContent eventId={eventId} onClose={onClose} />
        </Suspense>
      </DialogContent>
    </Dialog>
  )
}
