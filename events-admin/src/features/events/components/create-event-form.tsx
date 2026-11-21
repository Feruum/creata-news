import { Code, ConnectError, createClient } from '@connectrpc/connect'
import { useMutation, useSuspenseQuery, useTransport } from '@connectrpc/connect-query'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  createEvent,
  EventsService,
  type GetEventPublishReadinessResponse,
  LocationType,
  listCategories,
  listOrganizations,
  EventStatus as ProtoEventStatus,
  RoomsService,
  updateEvent
} from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'
import { Loader2, PencilIcon } from 'lucide-react'
import type { Value } from 'platejs'
import { Suspense, useEffect, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { bookingConflictsKey, bookingRequestsListKey } from '@/features/bookings'
import { clearServerError, handleServerErrors } from '@/shared/lib/form-utils'
import { EMPTY_PLATE_VALUE, getPlateTextLength, parsePlateJson } from '@/shared/lib/plate-utils'
import type { RequestErrorContext } from '@/shared/lib/request-errors'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/shared/ui/components/form'
import { Input } from '@/shared/ui/components/input'
import { cn } from '@/shared/ui/lib/utils'
import { eventDetailKey, eventPublishReadinessKey, eventsListKey } from '../api/keys'
import type { CreateEventFormData, CreateEventFormInput } from '../schema'
import { createEventSchema, defaultValues } from '../schema'
import { buildBadgeRequestChanges } from '../utils/event-badges'
import {
  buildEventLocation,
  buildEventSchedule,
  canRequestBooking,
  eventBookingProposalSignature,
  needsPublishedBookingCheck,
  readinessMessage
} from '../utils/event-submission'
import { finishCampusDraft, getSubmitIntent } from '../utils/event-workflows'
import { BadgeRequestSection } from './badge-request-section'
import { DescriptionEditorDialog } from './description-editor-dialog'
import { LocationFormSection } from './location-form-section'
import { OrganizationCategorySection } from './organization-category-section'
import { PlateEditor } from './plate-editor'
import { ScheduleSection } from './schedule-section'
import { SectionHeader } from './section-header'

// ============================================================================
// COMPONENT
// ============================================================================

interface CreateEventFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  eventId?: number
  initialValues?: Partial<CreateEventFormData>
  onCreatedDraft?: (eventId: number) => void
}

export function CreateEventForm({ open, onOpenChange, eventId, initialValues, onCreatedDraft }: CreateEventFormProps) {
  const queryClient = useQueryClient()
  const transport = useTransport()
  const eventsClient = useMemo(() => createClient(EventsService, transport), [transport])
  const roomsClient = useMemo(() => createClient(RoomsService, transport), [transport])
  const isEditMode = !!eventId
  const [isDescriptionEditorOpen, setIsDescriptionEditorOpen] = useState(false)
  const [isWorking, setIsWorking] = useState(false)
  const submissionLock = useRef(false)
  const createdDraftId = useRef<number | undefined>(undefined)
  const retryCreatedDraft = useRef<(() => Promise<boolean>) | undefined>(undefined)
  const [savedDraftId, setSavedDraftId] = useState<number>()
  const [bookingReviewResult, setBookingReviewResult] = useState<{
    signature: string
    readiness: GetEventPublishReadinessResponse
  } | null>(null)

  const form = useForm<CreateEventFormInput, unknown, CreateEventFormData>({
    resolver: zodResolver(createEventSchema),
    defaultValues: { ...defaultValues, ...initialValues },
    mode: 'onBlur'
  })

  const { isDirty, dirtyFields } = form.formState
  const proposalSignature = eventBookingProposalSignature(form.watch())
  const bookingReview = bookingReviewResult?.signature === proposalSignature ? bookingReviewResult.readiness : null
  const setBookingReview = (readiness: GetEventPublishReadinessResponse | null) => {
    setBookingReviewResult(readiness ? { signature: proposalSignature, readiness } : null)
  }
  useEffect(() => {
    setBookingReviewResult((current) => (current?.signature === proposalSignature ? current : null))
  }, [proposalSignature])
  const watchDescriptionJson = form.watch('description')
  const watchDescriptionValue = parsePlateJson(watchDescriptionJson)

  const organizationsQuery = useSuspenseQuery(listOrganizations, { page: 1, limit: 1000 })
  const organizations = organizationsQuery.data.organizations

  const categoriesQuery = useSuspenseQuery(listCategories, { page: 1, limit: 1000 })
  const categories = categoriesQuery.data.categories

  // The form owns all failures from this workflow, including uploads and preflight.
  const createMutation = useMutation(createEvent, { meta: { errorToast: false } })
  const updateMutation = useMutation(updateEvent, { meta: { errorToast: false } })
  const isSubmitting = isWorking || form.formState.isSubmitting

  const invalidateEvent = async (id: number) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: eventsListKey() }),
      queryClient.invalidateQueries({ queryKey: eventDetailKey(id) }),
      queryClient.invalidateQueries({ queryKey: eventPublishReadinessKey({ eventId: id }) })
    ])
  }

  const invalidateBookings = async (id: number) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: bookingRequestsListKey() }),
      queryClient.invalidateQueries({ queryKey: bookingConflictsKey() }),
      queryClient.invalidateQueries({ queryKey: eventPublishReadinessKey({ eventId: id }) })
    ])
  }

  const handleSubmit =
    (savedAsDraft: boolean, bookingAction?: 'request' | 'refresh') =>
    async (submitEvent?: React.BaseSyntheticEvent) => {
      submitEvent?.preventDefault()
      // A ref closes the same-tick gap before React renders disabled controls.
      if (submissionLock.current) return
      submissionLock.current = true
      setIsWorking(true)
      form.clearErrors('root')
      let context: RequestErrorContext = 'save'
      let refreshProposalReadiness: (() => Promise<GetEventPublishReadinessResponse>) | undefined
      try {
        await form.handleSubmit(async (data) => {
          const structuredLocation = buildEventLocation(data)
          const schedule = buildEventSchedule(data)
          const existingId = eventId ?? createdDraftId.current

          if (eventId) {
            const current = (await eventsClient.getEvent({ id: eventId })).event
            if (!current) throw new ConnectError('Event missing', Code.NotFound)
            // The time picker displays minutes. Untouched values must retain saved seconds.
            if (!dirtyFields.hasTime && !dirtyFields.startDate && !dirtyFields.startTime) {
              schedule.startTime = current.startTime
            }
            if (!dirtyFields.hasTime && !dirtyFields.endDate && !dirtyFields.endTime) {
              schedule.endTime = current.endTime || current.startTime
            }
            if (needsPublishedBookingCheck(current, { ...schedule, structuredLocation })) {
              context = 'booking'
              const readinessInput = {
                eventId,
                proposedStartTime: schedule.startTime,
                proposedEndTime: schedule.endTime,
                proposedLocation: structuredLocation
              }
              const checkReadiness = () =>
                queryClient.fetchQuery({
                  queryKey: eventPublishReadinessKey(readinessInput),
                  queryFn: () => eventsClient.getEventPublishReadiness(readinessInput),
                  staleTime: 0,
                  retry: false,
                  meta: { errorToast: false }
                })
              refreshProposalReadiness = checkReadiness
              let readiness = await checkReadiness()
              setBookingReview(readiness)
              if (bookingAction === 'request' && canRequestBooking(readiness.state) && structuredLocation.roomId) {
                try {
                  await roomsClient.createBookingRequest({
                    eventId,
                    roomId: structuredLocation.roomId,
                    title: data.title,
                    organizationId: data.organizationIds[0],
                    ...schedule
                  })
                } catch (error) {
                  if (!(error instanceof ConnectError && error.code === Code.AlreadyExists)) throw error
                } finally {
                  await invalidateBookings(eventId)
                  readiness = await checkReadiness()
                  setBookingReview(readiness)
                }
              }
              // Booking actions only review/request a proposal. Saving remains explicit.
              if (bookingAction || !readiness.canPublishNow) return
            } else {
              setBookingReview(null)
            }
          }

          if (bookingAction) return
          context = 'upload'
          let imageUrl: string | undefined
          if (data.coverImage instanceof File) {
            const uploaded = await eventsClient.uploadEventImage({
              eventId: existingId ?? 0,
              organizationId: existingId ? 0 : (data.organizationIds[0] ?? 0),
              image: new Uint8Array(await data.coverImage.arrayBuffer()),
              contentType: data.coverImage.type
            })
            imageUrl = uploaded.publicUrl
          }
          context = 'save'
          const input = {
            title: data.title,
            description: data.description,
            ...schedule,
            organizationId: data.organizationIds[0],
            structuredLocation,
            ...buildBadgeRequestChanges(data, dirtyFields),
            ...(imageUrl ? { imageUrl } : {}),
            ...(data.categoryId ? { categoryId: data.categoryId } : {})
          }
          if (existingId) {
            await updateMutation.mutateAsync({ id: existingId, ...input })
            if (eventId) {
              await invalidateEvent(eventId)
              toast.success('Event updated!')
              onOpenChange(false)
              form.reset()
              return
            }
          } else {
            const needsRoomApproval =
              !!structuredLocation.roomId &&
              (data.locationType === LocationType.CAMPUS_ROOM || data.locationType === LocationType.HYBRID)
            const created = await createMutation.mutateAsync({
              ...input,
              userId: 0,
              socGpaRequested: data.socGpaRequested,
              irosRequested: data.irosRequested,
              ...(savedAsDraft || needsRoomApproval ? { status: ProtoEventStatus.DRAFT } : {})
            })
            if (!created.event?.id) throw new Error('Event response missing')
            createdDraftId.current = created.event.id
          }

          const id = createdDraftId.current!
          const needsRoomApproval =
            !!structuredLocation.roomId &&
            (data.locationType === LocationType.CAMPUS_ROOM || data.locationType === LocationType.HYBRID)
          if (!savedAsDraft && needsRoomApproval) {
            const completeDraft = async () => {
              const outcome = await finishCampusDraft({
                requestBooking: data.requestRoomBooking
                  ? () =>
                      roomsClient.createBookingRequest({
                        eventId: id,
                        roomId: structuredLocation.roomId,
                        organizationId: data.organizationIds[0],
                        title: data.title,
                        ...schedule
                      })
                  : undefined,
                checkReadiness: () => eventsClient.getEventPublishReadiness({ eventId: id }),
                publish: () => eventsClient.publishEvent({ eventId: id })
              })
              await invalidateBookings(id)
              await invalidateEvent(id)
              if (outcome === 'booking-failed' || outcome === 'publish-failed') {
                setSavedDraftId(id)
                onCreatedDraft?.(id)
                form.setError('root', {
                  type: 'server',
                  message:
                    outcome === 'booking-failed'
                      ? 'Draft saved. Room booking request failed. Retry to continue with this draft.'
                      : 'Draft saved. The event could not be published. Retry to continue with this draft.'
                })
                return false
              }
              if (outcome === 'published') {
                toast.success('Event published with an approved room booking!')
              } else {
                toast.warning(
                  data.requestRoomBooking
                    ? 'Draft saved. Waiting for room approval.'
                    : 'Draft saved. Room approval is required before publishing.',
                  {
                    description: (
                      <a className='underline' href={`${import.meta.env.BASE_URL}events?event=${id}`}>
                        Open draft to review or edit
                      </a>
                    ),
                    duration: Infinity
                  }
                )
                onCreatedDraft?.(id)
              }
              return true
            }
            retryCreatedDraft.current = completeDraft
            if (!(await completeDraft())) return
          } else {
            await invalidateEvent(id)
            toast.success(savedAsDraft ? 'Event saved as draft!' : 'Event created!')
          }
          createdDraftId.current = undefined
          retryCreatedDraft.current = undefined
          setSavedDraftId(undefined)
          onOpenChange(false)
          form.reset()
        })(submitEvent)
      } catch (error) {
        if (refreshProposalReadiness && error instanceof ConnectError && error.code === Code.FailedPrecondition) {
          context = 'booking'
          try {
            const readiness = await refreshProposalReadiness()
            setBookingReview(readiness)
            if (!readiness.canPublishNow) return
          } catch {
            // Preserve the original save failure if a status refresh also fails.
          }
        }
        handleServerErrors(form, error, {
          showToast: false,
          context,
          fieldMapping: {
            start_time: 'startTime',
            end_time: 'endTime',
            organization_id: 'organizationIds',
            category_id: 'categoryId',
            room_id: 'roomId',
            'structured_location.room_id': 'roomId',
            'structuredLocation.roomId': 'roomId',
            online_url: 'onlineUrl',
            external_address: 'externalAddress',
            image: 'coverImage',
            content_type: 'coverImage'
          }
        })
      } finally {
        submissionLock.current = false
        setIsWorking(false)
      }
    }
  const handleRetryDraft = async () => {
    const completeDraft = retryCreatedDraft.current
    if (!completeDraft || submissionLock.current) return
    submissionLock.current = true
    setIsWorking(true)
    form.clearErrors('root')
    try {
      if (await completeDraft()) {
        createdDraftId.current = undefined
        retryCreatedDraft.current = undefined
        setSavedDraftId(undefined)
        onOpenChange(false)
        form.reset()
      }
    } catch (error) {
      handleServerErrors(form, error, { showToast: false, context: 'booking' })
    } finally {
      submissionLock.current = false
      setIsWorking(false)
    }
  }

  const handleOpenDescriptionEditor = () => {
    setIsDescriptionEditorOpen(true)
  }

  const handleSaveDescription = (value: Value) => {
    form.setValue('description', JSON.stringify(value), { shouldValidate: true, shouldDirty: true })
    setIsDescriptionEditorOpen(false)
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen) => {
          if (!isSubmitting) onOpenChange(nextOpen)
        }}
      >
        <DialogContent
          className='flex max-h-[90vh] w-[min(900px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
          style={{ overscrollBehavior: 'contain' }}
          showCloseButton={false}
        >
          <DialogHeader className='shrink-0 border-b px-6 py-4'>
            <DialogTitle>{isEditMode ? 'Edit Event' : 'Create Event'}</DialogTitle>
          </DialogHeader>

          <Form {...form}>
            <form
              onSubmit={(event) => {
                const submitter = (event.nativeEvent as SubmitEvent).submitter
                const intent = getSubmitIntent(submitter instanceof HTMLButtonElement ? submitter : null)
                void handleSubmit(intent === 'draft')(event)
              }}
              onChange={() => {
                form.clearErrors('root')
                setBookingReview(null)
              }}
              aria-busy={isSubmitting}
              className='flex min-h-0 flex-1 flex-col overflow-hidden'
            >
              <div className='flex-1 overflow-y-auto'>
                <fieldset
                  disabled={isSubmitting || savedDraftId !== undefined}
                  className='grid min-w-0 gap-6 p-6 md:grid-cols-2'
                >
                  <legend className='sr-only'>Event details</legend>
                  {/* LEFT COLUMN: Basic Info + Schedule */}
                  <div className='space-y-6'>
                    {/* Basic Info Section */}
                    <fieldset className='space-y-4 rounded-lg border bg-muted/30 p-4'>
                      <legend className='sr-only'>Basic Information</legend>
                      <SectionHeader title='Basic Information' />

                      <FormField
                        control={form.control}
                        name='title'
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Title</FormLabel>
                            <FormControl>
                              <Input
                                placeholder='Annual Tech Conference…'
                                {...field}
                                onChange={(e) => {
                                  field.onChange(e)
                                  clearServerError(form, 'title')
                                }}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      {/* Description - Clickable preview that opens editor popup */}
                      <FormField
                        control={form.control}
                        name='description'
                        render={({ fieldState }) => (
                          <FormItem>
                            <FormLabel>Description</FormLabel>
                            <FormControl>
                              <button
                                type='button'
                                onClick={handleOpenDescriptionEditor}
                                className={cn(
                                  'group relative w-full rounded-lg border bg-background p-3 text-left transition-colors hover:bg-muted/50 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
                                  fieldState.invalid && 'border-destructive'
                                )}
                              >
                                {watchDescriptionJson ? (
                                  <div className='line-clamp-3 pointer-events-none'>
                                    <PlateEditor value={watchDescriptionValue} onChange={() => {}} readOnly />
                                  </div>
                                ) : (
                                  <span className='text-sm text-muted-foreground'>Click to add description…</span>
                                )}
                                <div className='absolute right-2 top-2 rounded-md bg-muted p-1 opacity-0 transition-opacity group-hover:opacity-100'>
                                  <PencilIcon className='h-3.5 w-3.5 text-muted-foreground' aria-hidden='true' />
                                </div>
                              </button>
                            </FormControl>
                            <p className='text-xs text-muted-foreground'>
                              {getPlateTextLength(watchDescriptionValue)}/2000
                            </p>
                            <FormMessage />
                          </FormItem>
                        )}
                      />

                      <LocationFormSection form={form} />
                    </fieldset>

                    <ScheduleSection form={form} />
                  </div>

                  <OrganizationCategorySection form={form} organizations={organizations} categories={categories} />
                  <BadgeRequestSection disabled={isSubmitting} />
                </fieldset>
                {form.formState.errors.root?.message && (
                  <p role='alert' className='px-6 pb-4 text-sm text-destructive'>
                    {form.formState.errors.root.message}
                  </p>
                )}
                {savedDraftId && (
                  <div className='space-y-2 px-6 pb-4 text-sm'>
                    <p role='status'>
                      Your draft is saved. Retry room booking to continue, or open the draft to edit it.
                    </p>
                    <a className='underline' href={`${import.meta.env.BASE_URL}events?event=${savedDraftId}`}>
                      Open draft to review or edit
                    </a>
                    <Button
                      type='button'
                      variant='outline'
                      disabled={isSubmitting}
                      onClick={() => void handleRetryDraft()}
                    >
                      Retry room booking
                    </Button>
                  </div>
                )}
                {bookingReview && (
                  <section
                    aria-label='Proposed room approval'
                    className='mx-6 mb-4 space-y-3 rounded-lg border p-4 text-sm'
                  >
                    <p role='status'>{readinessMessage(bookingReview)}</p>
                    {bookingReview.rejectionReason && <p>{bookingReview.rejectionReason}</p>}
                    <div className='flex flex-wrap gap-2'>
                      {canRequestBooking(bookingReview.state) && (
                        <Button
                          type='button'
                          disabled={isSubmitting}
                          onClick={() => void handleSubmit(false, 'request')()}
                        >
                          Request room approval
                        </Button>
                      )}
                      <Button
                        type='button'
                        variant='outline'
                        disabled={isSubmitting}
                        onClick={() => void handleSubmit(false, 'refresh')()}
                      >
                        Refresh room approval
                      </Button>
                    </div>
                  </section>
                )}
              </div>

              <DialogFooter className='shrink-0 border-t px-6 py-4'>
                <Button type='button' variant='outline' onClick={() => onOpenChange(false)} disabled={isSubmitting}>
                  Cancel
                </Button>
                {!isEditMode && (
                  <Button
                    type='button'
                    onClick={() => {
                      void handleSubmit(true)()
                    }}
                    variant='ghost'
                    disabled={isSubmitting || savedDraftId !== undefined || (!isDirty && !isEditMode)}
                  >
                    {isSubmitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
                    Save as Draft
                  </Button>
                )}
                <Button
                  type='submit'
                  name='intent'
                  value='publish'
                  disabled={isSubmitting || savedDraftId !== undefined || (!isDirty && !isEditMode)}
                >
                  {isSubmitting && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
                  {isEditMode ? 'Save Changes' : 'Create Event'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      <DescriptionEditorDialog
        open={isDescriptionEditorOpen}
        onOpenChange={setIsDescriptionEditorOpen}
        initialValue={watchDescriptionValue.length > 0 ? watchDescriptionValue : EMPTY_PLATE_VALUE}
        onSave={handleSaveDescription}
      />
    </>
  )
}

// ============================================================================
// TRIGGER WRAPPER COMPONENT
// ============================================================================

interface CreateEventTriggerProps {
  children: React.ReactNode
  startDate?: Date
  startTime?: { hour: number; minute: number }
}

export function CreateEventTrigger({ children, startDate, startTime }: CreateEventTriggerProps) {
  const [isOpen, setIsOpen] = useState(false)

  const initialValues = useMemo(() => {
    const values: Partial<CreateEventFormData> = {}

    if (startDate) {
      values.startDate = startDate
      values.endDate = startDate
    }

    if (startTime) {
      const startTimeStr = `${startTime.hour.toString().padStart(2, '0')}:${startTime.minute.toString().padStart(2, '0')}`
      const endHour = startTime.minute >= 30 ? startTime.hour + 1 : startTime.hour
      const endMinute = startTime.minute >= 30 ? 0 : 30
      const endTimeStr = `${endHour.toString().padStart(2, '0')}:${endMinute.toString().padStart(2, '0')}`
      values.startTime = startTimeStr
      values.endTime = endTimeStr
      values.hasTime = true
    }

    return values
  }, [startDate, startTime])

  return (
    <>
      <button type='button' onClick={() => setIsOpen(true)} className='w-full text-left'>
        {children}
      </button>
      <Suspense fallback={null}>
        <CreateEventForm open={isOpen} onOpenChange={setIsOpen} initialValues={initialValues} />
      </Suspense>
    </>
  )
}
