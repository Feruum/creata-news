import { getActorName } from './actor-name'
import { Suspense } from 'react'

import { useEntityActivityLog } from '../api/queries'
import { ActionDisplay } from './action-display'

function parseSnapshot(json: string): Record<string, unknown> {
  try {
    return JSON.parse(json)
  } catch {
    return {}
  }
}

function formatTime(iso: string): string {
  const date = new Date(iso)
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function ActivityTimelineContent({
  entityType,
  entityId
}: {
  entityType: string
  entityId: number
}) {
  const { data } = useEntityActivityLog({
    entityType,
    entityId,
    page: 1,
    limit: 50
  })

  const entries = data?.entries ?? []

  if (entries.length === 0) {
    return <p className='text-sm text-muted-foreground py-4'>No activity recorded yet.</p>
  }

  return (
    <div className='space-y-3'>
      {entries.map((entry) => {
        const actorSnapshot = parseSnapshot(entry.actorSnapshot)
        const actorName = getActorName(actorSnapshot, entry.actorId)

        return (
          <div key={entry.id.toString()} className='flex items-start gap-3'>
            <div className='mt-0.5'>
              <ActionDisplay action={entry.action} />
            </div>
            <div className='flex-1 min-w-0'>
              <span className='text-sm'>
                <span className='font-medium'>{actorName}</span>
              </span>
              <span className='text-xs text-muted-foreground ml-2'>
                {formatTime(entry.createdAt)}
              </span>
            </div>
          </div>
        )
      })}
    </div>
  )
}

export function ActivityTimeline({
  entityType,
  entityId
}: {
  entityType: string
  entityId: number
}) {
  return (
    <Suspense fallback={<div className='animate-pulse space-y-3'><div className='h-8 bg-muted rounded' /><div className='h-8 bg-muted rounded' /><div className='h-8 bg-muted rounded' /></div>}>
      <ActivityTimelineContent entityType={entityType} entityId={entityId} />
    </Suspense>
  )
}
