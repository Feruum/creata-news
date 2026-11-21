import { Loader2 } from 'lucide-react'
import { type RefObject, useState } from 'react'
import { toast } from 'sonner'
import { formatDate } from '@/shared/lib/utils'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger
} from '@/shared/ui/components/alert-dialog'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/shared/ui/components/dialog'
import { Label } from '@/shared/ui/components/label'
import { Textarea } from '@/shared/ui/components/textarea'
import { ScheduleApiError, scheduleErrorMessage } from '../api/client'
import { useReviewSchedule } from '../api/mutations'
import { useScheduleSubmission } from '../api/queries'
import type { ScheduleSettings, SubmissionDetails } from '../types'
import { ScheduleLessonList } from './schedule-lesson-list'
import { SubmissionStatusBadge } from './submission-status-badge'

interface ScheduleReviewDialogProps {
  id: string
  onClose: () => void
  returnFocusRef: RefObject<HTMLButtonElement | null>
}

export function ScheduleReviewDialog({ id, onClose, returnFocusRef }: ScheduleReviewDialogProps) {
  const query = useScheduleSubmission(id)
  const mutation = useReviewSchedule()
  const [reason, setReason] = useState('')
  const [approvalOpen, setApprovalOpen] = useState(false)
  const blockedDecision = mutation.error instanceof ScheduleApiError && [401, 403, 409].includes(mutation.error.status)
  const details = query.data
  const canDecide = details?.submission.status === 'pending' && !blockedDecision && !query.isError
  const busy = mutation.isPending || query.isFetching

  const decide = async (decision: 'approve' | 'reject') => {
    if (!canDecide || busy || (decision === 'reject' && !reason.trim())) return
    try {
      await mutation.mutateAsync({ id, decision, ...(decision === 'reject' ? { reason: reason.trim() } : {}) })
      toast.success(decision === 'approve' ? 'Schedule approved and published' : 'Schedule rejected')
      setApprovalOpen(false)
      onClose()
    } catch {
      setApprovalOpen(false)
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) onClose()
      }}
    >
      <DialogContent
        className='max-h-[90svh] min-w-0 overflow-y-auto sm:max-w-5xl'
        showCloseButton={!mutation.isPending}
        finalFocus={returnFocusRef}
      >
        <DialogHeader>
          <DialogTitle>Schedule review{details ? `: ${details.submission.groupName}` : ''}</DialogTitle>
          <DialogDescription>Compare the submitted schedule with the currently published schedule.</DialogDescription>
        </DialogHeader>
        {query.isPending && (
          <p role='status' className='text-sm text-muted-foreground'>
            Loading schedule details...
          </p>
        )}
        {query.isError ? (
          <div role='alert' className='space-y-3 text-sm'>
            <p>{scheduleErrorMessage(query.error)}</p>
            <Button variant='outline' disabled={query.isFetching} onClick={() => void query.refetch()}>
              Retry
            </Button>
          </div>
        ) : (
          details && <ScheduleComparison details={details} />
        )}
        {mutation.isError && (
          <p role='alert' className='rounded-lg border border-destructive/50 p-3 text-sm'>
            {scheduleErrorMessage(mutation.error)}
          </p>
        )}
        {canDecide && (
          <div className='space-y-2 border-t pt-4'>
            <Label htmlFor={`rejection-reason-${id}`}>Rejection reason</Label>
            <Textarea
              id={`rejection-reason-${id}`}
              aria-describedby={`rejection-help-${id}`}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder='Explain which lessons or settings need correction...'
              rows={3}
              disabled={busy}
            />
            <p id={`rejection-help-${id}`} className='text-xs text-muted-foreground'>
              Required to reject. The reason is saved with the review.
            </p>
          </div>
        )}
        <DialogFooter>
          <Button variant='outline' disabled={mutation.isPending} onClick={onClose}>
            Close review
          </Button>
          {canDecide && (
            <>
              <Button variant='destructive' disabled={busy || !reason.trim()} onClick={() => void decide('reject')}>
                {mutation.isPending && <Loader2 aria-hidden='true' className='size-4 animate-spin' />}
                Reject
              </Button>
              <AlertDialog open={approvalOpen} onOpenChange={setApprovalOpen}>
                <AlertDialogTrigger render={<Button disabled={busy} />}>Approve</AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>Approve schedule for {details?.submission.groupName}?</AlertDialogTitle>
                    <AlertDialogDescription>
                      Approval publishes the complete submitted schedule and replaces the currently published schedule
                      for this group.
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel disabled={mutation.isPending}>Cancel</AlertDialogCancel>
                    <Button disabled={busy} onClick={() => void decide('approve')}>
                      {mutation.isPending && <Loader2 aria-hidden='true' className='size-4 animate-spin' />}
                      Approve and publish
                    </Button>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function ScheduleComparison({ details }: { details: SubmissionDetails }) {
  const { submission, published, diff } = details
  const settingsChanged =
    published &&
    (published.settings.studyYear !== submission.settings.studyYear ||
      published.settings.term !== submission.settings.term ||
      published.settings.firstWeekStart !== submission.settings.firstWeekStart)

  return (
    <div className='min-w-0 space-y-6'>
      <dl className='grid gap-3 rounded-lg bg-muted/50 p-4 text-sm sm:grid-cols-2'>
        <div>
          <dt className='text-muted-foreground'>Status</dt>
          <dd className='mt-1'>
            <SubmissionStatusBadge status={submission.status} />
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground'>Submitted</dt>
          <dd>
            <time dateTime={submission.createdAt}>{formatDate(submission.createdAt, true)}</time>
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground'>Author</dt>
          <dd className='break-all'>{submission.authorId}</dd>
        </div>
        {submission.reviewedAt && (
          <div>
            <dt className='text-muted-foreground'>Reviewed</dt>
            <dd>
              <time dateTime={submission.reviewedAt}>{formatDate(submission.reviewedAt, true)}</time>
            </dd>
          </div>
        )}
        {submission.reviewerId && (
          <div>
            <dt className='text-muted-foreground'>Reviewer</dt>
            <dd className='break-all'>{submission.reviewerId}</dd>
          </div>
        )}
        {submission.rejectionReason && (
          <div className='sm:col-span-2'>
            <dt className='text-muted-foreground'>Rejection reason</dt>
            <dd className='whitespace-pre-wrap'>{submission.rejectionReason}</dd>
          </div>
        )}
      </dl>
      <div className='grid gap-4 sm:grid-cols-2'>
        <ScheduleSettingsCard title='Submitted settings' settings={submission.settings} />
        {published ? (
          <ScheduleSettingsCard title='Published settings' settings={published.settings} />
        ) : (
          <section className='rounded-lg border p-4 text-sm'>
            <h3 className='font-medium'>Published schedule</h3>
            <p className='mt-2 text-muted-foreground'>No schedule has been published for this group.</p>
          </section>
        )}
      </div>
      <section className='min-w-0 space-y-4' aria-label='Schedule changes'>
        <div>
          <h3 className='font-semibold'>Changes from the published schedule</h3>
          {settingsChanged && <p className='mt-1 text-sm'>Schedule settings changed.</p>}
          {!diff.added.length && !diff.removed.length && (
            <p className='mt-1 text-sm text-muted-foreground'>No lesson changes.</p>
          )}
        </div>
        <div className='min-w-0 space-y-2'>
          <h4 className='text-sm font-medium'>Added lessons ({diff.added.length})</h4>
          <ScheduleLessonList label='Added lessons' events={diff.added} />
        </div>
        <div className='min-w-0 space-y-2'>
          <h4 className='text-sm font-medium'>Removed lessons ({diff.removed.length})</h4>
          <ScheduleLessonList label='Removed lessons' events={diff.removed} />
        </div>
      </section>
      <section className='min-w-0 space-y-2' aria-label='Complete submitted snapshot'>
        <h3 className='font-semibold'>Complete submitted schedule ({submission.events.length} lessons)</h3>
        <ScheduleLessonList label='Submitted lessons' events={submission.events} />
      </section>
    </div>
  )
}

function ScheduleSettingsCard({ title, settings }: { title: string; settings: ScheduleSettings }) {
  return (
    <section className='rounded-lg border p-4 text-sm'>
      <h3 className='font-medium'>{title}</h3>
      <dl className='mt-2 grid grid-cols-2 gap-1'>
        <dt className='text-muted-foreground'>Study year</dt>
        <dd>{settings.studyYear}</dd>
        <dt className='text-muted-foreground'>Term</dt>
        <dd>{settings.term}</dd>
        <dt className='text-muted-foreground'>First week starts</dt>
        <dd>
          {formatDate(settings.firstWeekStart, { timeZone: 'UTC', month: 'short', day: 'numeric', year: 'numeric' })}
        </dd>
      </dl>
    </section>
  )
}
