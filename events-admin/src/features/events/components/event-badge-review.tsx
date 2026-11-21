'use client'

import { type Event, EventBadgeStatus } from '@studyverse/proto'
import { Button } from '@/shared/ui/components/button'
import { getBadgeStatusLabel, getEventBadgeEntries, isVisibleEventBadge } from '../utils/event-badges'

interface EventBadgeReviewProps {
  event: Pick<Event, 'socGpaBadge' | 'irosBadge'>
  canReview: boolean
  isPending?: boolean
  hasError?: boolean
  onReview: (badge: 'socGpa' | 'iros', status: EventBadgeStatus) => void
}

export function EventBadgeReview({
  event,
  canReview,
  isPending = false,
  hasError = false,
  onReview
}: EventBadgeReviewProps) {
  const badges = getEventBadgeEntries(event).filter(({ badge }) => isVisibleEventBadge(badge, true))

  return (
    <section className='space-y-3 rounded-lg border p-4' aria-label='Event badge approval'>
      <div>
        <h3 className='text-sm font-semibold'>Badge approval</h3>
        <p className='mt-1 text-xs text-muted-foreground'>
          Administrators review each badge. Participant credit is handled externally after approval.
        </p>
      </div>
      {!badges.length && <p className='text-sm text-muted-foreground'>No badges requested.</p>}
      {badges.map(({ key, label, badge }) => (
        <div key={key} className='flex flex-wrap items-center justify-between gap-2'>
          <div className='space-y-0.5'>
            <p className='text-sm font-medium'>{label}</p>
            <p className='text-xs text-muted-foreground'>{getBadgeStatusLabel(badge?.status)}</p>
          </div>
          {canReview && (
            <div className='flex flex-wrap gap-2'>
              {badge?.status !== EventBadgeStatus.APPROVED && (
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  aria-label={`Approve ${label}`}
                  disabled={isPending}
                  onClick={() => onReview(key, EventBadgeStatus.APPROVED)}
                >
                  Approve
                </Button>
              )}
              {badge?.status !== EventBadgeStatus.REJECTED && (
                <Button
                  type='button'
                  variant='ghost'
                  size='sm'
                  aria-label={
                    badge?.status === EventBadgeStatus.APPROVED ? `Remove ${label} approval` : `Reject ${label}`
                  }
                  disabled={isPending}
                  onClick={() => onReview(key, EventBadgeStatus.REJECTED)}
                >
                  {badge?.status === EventBadgeStatus.APPROVED ? 'Remove approval' : 'Reject'}
                </Button>
              )}
            </div>
          )}
        </div>
      ))}
      {hasError && (
        <p role='alert' className='text-sm text-destructive'>
          Could not save badge review. Please try again.
        </p>
      )}
    </section>
  )
}
