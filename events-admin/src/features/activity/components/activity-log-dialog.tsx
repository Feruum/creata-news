import { getActorName } from './actor-name'
import { useQuery } from '@connectrpc/connect-query'
import { listActivityLog } from '@studyverse/proto'
import { CalendarDays, Loader2 } from 'lucide-react'
import { useState } from 'react'

import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'

import { ACTION_MAP, type ActionConfig } from './action-display'
import { ActionFilterPopover, DateRangeFilter } from './activity-filters'
import {
  ActorAvatar,
  BADGE_COLOR_CLASSES,
  ENTITY_TYPE_ICONS,
  formatTimeAgo,
  getActionVerb,
  parseSnapshot
} from './activity-utils'

const PAGE_SIZE = 20

export function ActivityLogDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-2xl'>
        <DialogHeader>
          <DialogTitle>Activity Log</DialogTitle>
        </DialogHeader>
        <ActivityLogFeed />
      </DialogContent>
    </Dialog>
  )
}

function ActivityLogFeed() {
  const [actionFilters, setActionFilters] = useState<string[]>([])
  const [after, setAfter] = useState<string | null>(null)
  const [before, setBefore] = useState<string | null>(null)
  const [limit, setLimit] = useState(PAGE_SIZE)

  const { data, isFetching } = useQuery(listActivityLog, {
    actionFilters: actionFilters.length > 0 ? actionFilters : undefined,
    after: after ?? undefined,
    before: before ?? undefined,
    page: 1,
    limit
  })

  const entries = data?.entries ?? []
  const hasMore = entries.length >= limit

  const resetLimit = () => setLimit(PAGE_SIZE)

  return (
    <>
      <div className='flex items-center gap-2'>
        <ActionFilterPopover
          value={actionFilters}
          onChange={(v) => {
            setActionFilters(v)
            resetLimit()
          }}
        />
        <DateRangeFilter
          after={after}
          before={before}
          onAfterChange={(v) => {
            setAfter(v)
            resetLimit()
          }}
          onBeforeChange={(v) => {
            setBefore(v)
            resetLimit()
          }}
        />
      </div>
      <div className='max-h-[60vh] overflow-y-auto'>
        {entries.length === 0 && !isFetching ? (
          <p className='text-sm text-muted-foreground py-8 text-center'>No activity found</p>
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
                    <p className='text-sm leading-snug'>
                      <span className='font-medium'>{actorName}</span>{' '}
                      <span className='text-muted-foreground'>{getActionVerb(entry.action)}</span>{' '}
                      <span className='font-medium truncate'>{entityName}</span>
                    </p>
                    <div className='flex items-center gap-1 mt-0.5'>
                      <EntityIcon className='size-3 text-muted-foreground' />
                      <span className='text-xs text-muted-foreground capitalize'>
                        {entry.entityType} · {formatTimeAgo(entry.createdAt)}
                      </span>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
        {isFetching && (
          <div className='flex justify-center py-4'>
            <Loader2 className='size-5 animate-spin text-muted-foreground' />
          </div>
        )}
        {hasMore && !isFetching && (
          <div className='flex justify-center pt-3'>
            <Button variant='outline' size='sm' onClick={() => setLimit((l) => l + PAGE_SIZE)}>
              Load more
            </Button>
          </div>
        )}
      </div>
    </>
  )
}
