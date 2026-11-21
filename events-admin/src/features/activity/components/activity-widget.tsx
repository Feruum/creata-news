import { getActorName } from './actor-name'
import { CalendarDays, VenetianMask } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/components/button'

import { useActivityLogLatest } from '../api/queries'
import { ACTION_MAP, type ActionConfig } from './action-display'
import { ActivityLogDialog } from './activity-log-dialog'
import {
  ActorAvatar,
  BADGE_COLOR_CLASSES,
  ENTITY_TYPE_ICONS,
  formatTimeAgo,
  getActionVerb,
  parseSnapshot
} from './activity-utils'

export function ActivityWidget({ className, organizationIds }: { className?: string; organizationIds?: number[] }) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const { data } = useActivityLogLatest({
    organizationIds,
    limit: 6
  })

  const entries = data?.entries ?? []

  return (
    <div className={cn('rounded-xl border bg-card p-4 flex flex-col gap-3', className)}>
      <div className='flex items-center justify-between'>
        <div className='flex items-center gap-2'>
          <VenetianMask className='size-5' />
          <div>
            <h3 className='text-sm font-medium'>Team Activity</h3>
            <p className='text-xs text-muted-foreground mt-0.5'>Recent actions on platform</p>
          </div>
        </div>
        <Button variant='outline' size='sm' className="text-xs" onClick={() => setDialogOpen(true)}>
          See More
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className='text-xs text-muted-foreground py-4 text-center'>No recent activity</p>
      ) : (
        <div className='space-y-3'>
          {entries.map((entry) => {
            const actorSnapshot = parseSnapshot(entry.actorSnapshot)
            const entitySnapshot = parseSnapshot(entry.entitySnapshot)
            const actorName = getActorName(actorSnapshot, entry.actorId)
            const telegramUsername = actorSnapshot.telegram_username as string | undefined
            const entityName =
              (entitySnapshot.title as string) ||
              (entitySnapshot.name as string) ||
              `${entry.entityType}#${entry.entityId}`

            const actionConfig: ActionConfig = ACTION_MAP[entry.action] ?? {
              label: entry.action,
              icon: CalendarDays,
              color: 'gray' as const
            }
            const ActionIcon = actionConfig.icon
            const EntityIcon = ENTITY_TYPE_ICONS[entry.entityType] ?? CalendarDays

            return (
              <div key={entry.id.toString()} className='flex items-start gap-2.5'>
                <div className='relative shrink-0'>
                  <ActorAvatar name={actorName} telegramUsername={telegramUsername} />
                  <div className='absolute -bottom-0.5 -right-0.5 size-4 rounded-full bg-background border flex items-center justify-center'>
                    <ActionIcon className={`size-2.5 ${BADGE_COLOR_CLASSES[actionConfig.color]}`} />
                  </div>
                </div>
                <div className='flex-1 min-w-0'>
                  <p className='text-xs leading-snug'>
                    <span className='font-medium'>{actorName}</span>{' '}
                    <span className='text-muted-foreground'>{getActionVerb(entry.action)}</span>{' '}
                    <span className='font-medium truncate'>{entityName}</span>
                  </p>
                  <div className='flex items-center gap-1 mt-0.5'>
                    <EntityIcon className='size-2.5 text-muted-foreground' />
                    <span className='text-[10px] text-muted-foreground capitalize'>
                      {entry.entityType} · {formatTimeAgo(entry.createdAt)}
                    </span>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
      <ActivityLogDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  )
}
