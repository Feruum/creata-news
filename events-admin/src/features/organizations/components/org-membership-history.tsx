import type { MembershipHistoryEntry } from '@studyverse/proto'
import { CalendarDays, Loader2, UserMinus, UserPlus, Users } from 'lucide-react'
import { useState } from 'react'
import { ActionFilterPopover, DateRangeFilter } from '@/features/activity'
import { useOrganizationMembershipHistory } from '@/features/organizations/api/queries'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { Button } from '@/shared/ui/components/button'

const PAGE_SIZE = 20

// Membership-specific action groups for the filter popover
// The ActionFilterPopover uses ACTION_GROUPS from action-display which includes all actions.
// We need a custom filter that only shows membership-relevant actions.

const ACTION_ICON_MAP: Record<string, typeof UserPlus> = {
  assigned: UserPlus,
  role_changed: Users,
  removed: UserMinus,
  org_deleted: CalendarDays,
  org_archived: CalendarDays
}

const ACTION_VERB_MAP: Record<string, string> = {
  assigned: 'was assigned to',
  role_changed: 'role changed in',
  removed: 'was removed from',
  org_deleted: 'org was deleted',
  org_archived: 'org was archived'
}

function MembershipAvatar({ name, avatarUrl }: { name: string; avatarUrl?: string }) {
  const [imageError, setImageError] = useState(false)
  const fallbackUrl = `https://api.dicebear.com/9.x/glass/svg?seed=${encodeURIComponent(name)}`

  if (!avatarUrl || imageError) {
    return <img alt={name} className='size-7 shrink-0 rounded-full border' src={fallbackUrl} />
  }

  return (
    <img
      alt={name}
      className='size-7 shrink-0 rounded-full border'
      onError={() => setImageError(true)}
      src={avatarUrl}
    />
  )
}

function formatTimeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'yesterday'
  return `${days}d ago`
}

function EntryDescription({ entry }: { entry: MembershipHistoryEntry }) {
  const userName = (
    <>
      {entry.userDisplayName ?? `User #${entry.userId}`}{' '}
      <AchievementNameBadge badge={entry.selectedBadge} userId={entry.userId} />
    </>
  )
  const verb = ACTION_VERB_MAP[entry.action] ?? entry.action
  const roleName = entry.roleName

  switch (entry.action) {
    case 'assigned':
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span> <span className='text-muted-foreground'>was assigned</span>
          {roleName && (
            <>
              {' '}
              <span className='text-muted-foreground'>as</span> <span className='font-medium'>{roleName}</span>
            </>
          )}
        </p>
      )
    case 'role_changed':
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span> <span className='text-muted-foreground'>role changed</span>
          {entry.previousRoleName && (
            <>
              {' '}
              <span className='text-muted-foreground'>from</span>{' '}
              <span className='font-medium'>{entry.previousRoleName}</span>
            </>
          )}
          {roleName && (
            <>
              {' '}
              <span className='text-muted-foreground'>to</span> <span className='font-medium'>{roleName}</span>
            </>
          )}
        </p>
      )
    case 'removed':
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span> <span className='text-muted-foreground'>was removed</span>
          {entry.previousRoleName && (
            <>
              {' '}
              <span className='text-muted-foreground'>from</span>{' '}
              <span className='font-medium'>{entry.previousRoleName}</span>
            </>
          )}
        </p>
      )
    case 'org_deleted':
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span>{' '}
          <span className='text-muted-foreground'>membership ended — org deleted</span>
        </p>
      )
    case 'org_archived':
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span>{' '}
          <span className='text-muted-foreground'>membership archived — org archived</span>
        </p>
      )
    default:
      return (
        <p className='text-sm leading-snug'>
          <span className='font-medium'>{userName}</span> <span className='text-muted-foreground'>{verb}</span>
        </p>
      )
  }
}

export function OrgMembershipHistory({ organizationId }: { organizationId: number }) {
  const [actionFilters, setActionFilters] = useState<string[]>([])
  const [after, setAfter] = useState<string | null>(null)
  const [before, setBefore] = useState<string | null>(null)
  const [limit, setLimit] = useState(PAGE_SIZE)

  const actionFilter = actionFilters.length > 0 ? actionFilters.join(',') : undefined

  const { data, isFetching, isError, refetch } = useOrganizationMembershipHistory({
    organizationId,
    actionFilter,
    limit,
    offset: 0
  })

  const entries = data?.entries ?? []

  const filteredEntries = entries.filter((entry) => {
    if (!after && !before) return true

    const entryTime = new Date(entry.effectiveAt).getTime()

    if (after) {
      const afterTime = new Date(after).getTime()
      if (entryTime < afterTime) return false
    }

    if (before) {
      // Add 24 hours minus 1 ms to make it inclusive of the entire 'before' day
      const beforeTime = new Date(before).getTime() + 86399999
      if (entryTime > beforeTime) return false
    }

    return true
  })

  const hasMore = entries.length >= limit

  const resetLimit = () => setLimit(PAGE_SIZE)

  return (
    <div className='space-y-4'>
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

      <div className='max-h-[50vh] overflow-y-auto'>
        {isError ? (
          <div role='alert' className='space-y-3 py-8 text-sm text-muted-foreground'>
            <p>Could not load membership history.</p>
            <Button variant='outline' size='sm' onClick={() => refetch()} disabled={isFetching}>
              Try again
            </Button>
          </div>
        ) : filteredEntries.length === 0 && !isFetching ? (
          <p className='text-sm text-muted-foreground py-8 text-center'>
            {entries.length === 0 ? 'No membership changes recorded yet' : 'No changes match the selected date range'}
          </p>
        ) : (
          <div className='space-y-3'>
            {filteredEntries.map((entry) => {
              const ActionIcon = ACTION_ICON_MAP[entry.action] ?? CalendarDays

              return (
                <div key={entry.id.toString()} className='flex items-start gap-2.5'>
                  <div className='relative shrink-0'>
                    <MembershipAvatar
                      name={entry.userDisplayName ?? `User #${entry.userId}`}
                      avatarUrl={entry.userAvatarUrl}
                    />
                    <div className='absolute -bottom-0.5 -right-0.5 size-4 rounded-full bg-background border flex items-center justify-center'>
                      <ActionIcon className='size-2.5 text-muted-foreground' />
                    </div>
                  </div>
                  <div className='flex-1 min-w-0'>
                    <EntryDescription entry={entry} />
                    <div className='flex items-center gap-1 mt-0.5'>
                      {entry.performedByDisplayName && (
                        <span className='inline-flex items-center gap-1 text-xs text-muted-foreground'>
                          by {entry.performedByDisplayName}{' '}
                          {entry.performedBy && (
                            <AchievementNameBadge badge={entry.performedBySelectedBadge} userId={entry.performedBy} />
                          )}{' '}
                          ·{' '}
                        </span>
                      )}
                      <span className='text-xs text-muted-foreground'>{formatTimeAgo(entry.effectiveAt)}</span>
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
    </div>
  )
}
