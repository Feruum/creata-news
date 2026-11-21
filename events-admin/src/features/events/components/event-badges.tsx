'use client'

import { type Event, EventBadgeStatus } from '@studyverse/proto'
import { Badge } from '@/shared/ui/components/badge'
import { cn } from '@/shared/ui/lib/utils'
import { getBadgeStatusLabel, getEventBadgeEntries, isVisibleEventBadge } from '../utils/event-badges'

interface EventBadgesProps {
  event: Pick<Event, 'socGpaBadge' | 'irosBadge'>
  showRequests?: boolean
  className?: string
}

export function EventBadges({ event, showRequests = false, className }: EventBadgesProps) {
  const badges = getEventBadgeEntries(event).filter(({ badge }) => isVisibleEventBadge(badge, showRequests))
  if (!badges.length) return null

  return (
    <div className={cn('flex flex-wrap items-center gap-1.5', className)}>
      {badges.map(({ key, label, badge }) => {
        const approved = badge?.status === EventBadgeStatus.APPROVED
        const status = getBadgeStatusLabel(badge?.status)
        return (
          <Badge
            key={key}
            variant='outline'
            aria-label={`${label} ${status.toLowerCase()}`}
            className={cn(
              'text-xs',
              approved
                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
                : 'text-muted-foreground'
            )}
          >
            {label}
            {showRequests ? ` · ${status}` : ''}
          </Badge>
        )
      })}
    </div>
  )
}
