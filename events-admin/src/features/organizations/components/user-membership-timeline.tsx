import type { MembershipTenure } from '@studyverse/proto'
import { format, parseISO } from 'date-fns'
import { Building2Icon, ChevronDownIcon, ChevronRightIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Badge } from '@/shared/ui/components/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/shared/ui/components/collapsible'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { useUserMembershipTimeline } from '../api/queries'

interface UserMembershipTimelineProps {
  userId: number
}

export function UserMembershipTimeline({ userId }: UserMembershipTimelineProps) {
  const { data, isLoading, error } = useUserMembershipTimeline({ userId, includeEnded: true })

  const orgGroups = useMemo(() => {
    if (!data?.tenures) return []

    const groups = new Map<
      number | string,
      {
        organizationId?: number
        organizationTitle: string
        organizationImageUrl?: string
        isCurrent: boolean
        tenures: MembershipTenure[]
      }
    >()
    for (const tenure of data.tenures) {
      const key = tenure.organizationId || tenure.organizationTitle
      if (!key) continue

      const existing = groups.get(key)
      if (existing) {
        existing.tenures.push(tenure)
        if (tenure.isCurrent) {
          existing.isCurrent = true
        }
      } else {
        groups.set(key, {
          organizationId: tenure.organizationId,
          organizationTitle: tenure.organizationTitle,
          organizationImageUrl: tenure.organizationImageUrl,
          isCurrent: tenure.isCurrent,
          tenures: [tenure]
        })
      }
    }

    return Array.from(groups.values()).sort((a, b) => {
      if (a.isCurrent && !b.isCurrent) return -1
      if (!a.isCurrent && b.isCurrent) return 1
      return a.organizationTitle.localeCompare(b.organizationTitle)
    })
  }, [data?.tenures])

  const threshold = 3
  const hasMore = orgGroups.length > threshold
  const [isOpen, setIsOpen] = useState(false)

  if (isLoading) {
    return (
      <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
        <div className='flex items-center gap-2 mb-4'>
          <Skeleton className='h-6 w-40' />
          <Skeleton className='h-5 w-8 rounded-full' />
        </div>
        <div className='space-y-4'>
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-24 w-full' />
        </div>
      </div>
    )
  }

  if (error || !data) {
    return null
  }

  if (orgGroups.length === 0) {
    return (
      <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
        <div className='flex items-center gap-2 mb-4'>
          <h3 className='font-semibold text-lg'>Membership History</h3>
          <Badge variant='secondary' className='rounded-full'>
            0
          </Badge>
        </div>
        <div className='text-sm text-muted-foreground py-8 text-center border rounded-lg bg-muted/20'>
          No membership history found.
        </div>
      </div>
    )
  }

  const visibleGroups = orgGroups.slice(0, threshold)
  const hiddenGroups = orgGroups.slice(threshold)

  return (
    <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
      <div className='flex items-center gap-2 mb-4'>
        <h3 className='font-semibold text-lg'>Membership History</h3>
        <Badge variant='secondary' className='rounded-full'>
          {orgGroups.length}
        </Badge>
      </div>

      <div className='space-y-4'>
        <div className='space-y-3'>
          {visibleGroups.map((group) => (
            <OrgGroupCard key={group.organizationId || group.organizationTitle} group={group} />
          ))}
        </div>

        {hasMore && (
          <Collapsible open={isOpen} onOpenChange={setIsOpen} className='space-y-3'>
            <CollapsibleContent className='space-y-3'>
              {hiddenGroups.map((group) => (
                <OrgGroupCard key={group.organizationId || group.organizationTitle} group={group} />
              ))}
            </CollapsibleContent>
            <CollapsibleTrigger
              render={
                <button
                  type='button'
                  className='flex w-full items-center justify-center gap-2 py-2 text-sm text-muted-foreground hover:text-foreground transition-colors'
                />
              }
            >
              {isOpen ? (
                <>
                  Show less <ChevronDownIcon className='w-4 h-4' />
                </>
              ) : (
                <>
                  Show {hiddenGroups.length} more <ChevronRightIcon className='w-4 h-4' />
                </>
              )}
            </CollapsibleTrigger>
          </Collapsible>
        )}
      </div>
    </div>
  )
}

function OrgGroupCard({
  group
}: {
  group: {
    organizationId?: number
    organizationTitle: string
    organizationImageUrl?: string
    isCurrent: boolean
    tenures: MembershipTenure[]
  }
}) {
  return (
    <div className='flex gap-4 p-4 border rounded-lg bg-card'>
      <div className='shrink-0 mt-1'>
        {group.organizationImageUrl ? (
          <img
            src={group.organizationImageUrl}
            alt={group.organizationTitle}
            className='w-10 h-10 rounded-md object-cover border'
          />
        ) : (
          <div className='w-10 h-10 rounded-md bg-muted flex items-center justify-center border'>
            <Building2Icon className='w-5 h-5 text-muted-foreground' />
          </div>
        )}
      </div>
      <div className='flex-1 min-w-0'>
        <div className='flex items-center justify-between gap-2 mb-2'>
          <h4 className='font-medium truncate'>{group.organizationTitle}</h4>
          {group.isCurrent && (
            <Badge variant='secondary' className='shrink-0 bg-emerald-100 text-emerald-700 hover:bg-emerald-100'>
              Active
            </Badge>
          )}
        </div>
        <div className='space-y-2'>
          {group.tenures.map((tenure, idx) => (
            <div
              key={`${tenure.roleKey}-${tenure.startedAt}-${idx}`}
              className='text-sm flex flex-col sm:flex-row sm:items-center justify-between gap-1 sm:gap-4'
            >
              <span className='font-medium text-muted-foreground'>{tenure.roleName}</span>
              <span className='text-muted-foreground/80 text-xs tabular-nums shrink-0'>
                {formatDate(tenure.startedAt)} - {tenure.endedAt ? formatDate(tenure.endedAt) : 'Present'}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function formatDate(isoString: string): string {
  try {
    return format(parseISO(isoString), 'MMM yyyy')
  } catch {
    return 'Invalid Date'
  }
}
