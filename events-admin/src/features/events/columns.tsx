import type { Event } from '@studyverse/proto'
import type { ColumnDef, FilterFn } from '@tanstack/react-table'
import { CalendarDaysIcon, ImageIcon, MapPinIcon } from 'lucide-react'

import { EventStatusBadge } from '@/shared/ui/components/event-status-badge'
import { cn } from '@/shared/ui/lib/utils'
import { EventBadges } from './components/event-badges'
import { FormatBadge } from './components/format-badge'
import { getEventDescriptionText } from './utils/description-text'
import { getEventStatus } from './utils/event-status'
import { formatStructuredLocation } from './utils/format-location'

// ============================================================================
// Helper Functions
// ============================================================================

function getRelativeTime(dateString: string): { text: string; isUpcoming: boolean; isPast: boolean } {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.round(diffMs / (1000 * 60))
  const diffHours = Math.round(diffMs / (1000 * 60 * 60))
  const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24))

  const isUpcoming = diffMs > 0
  const isPast = diffMs < 0

  if (Math.abs(diffMins) < 60) {
    return {
      text: isUpcoming ? `in ${diffMins}m` : `${Math.abs(diffMins)}m ago`,
      isUpcoming,
      isPast
    }
  }
  if (Math.abs(diffHours) < 24) {
    return {
      text: isUpcoming ? `in ${diffHours}h` : `${Math.abs(diffHours)}h ago`,
      isUpcoming,
      isPast
    }
  }
  if (Math.abs(diffDays) < 7) {
    return {
      text: isUpcoming ? `in ${diffDays}d` : `${Math.abs(diffDays)}d ago`,
      isUpcoming,
      isPast
    }
  }
  if (Math.abs(diffDays) < 30) {
    const weeks = Math.round(Math.abs(diffDays) / 7)
    return {
      text: isUpcoming ? `in ${weeks}w` : `${weeks}w ago`,
      isUpcoming,
      isPast
    }
  }

  return { text: '', isUpcoming: false, isPast: true }
}

// ============================================================================
// Filter Functions
// ============================================================================
// Filter Functions
// ============================================================================

export const multiColumnFilterFn: FilterFn<Event> = (row, _columnId, filterValue) => {
  const location = formatStructuredLocation(row.original.structuredLocation)
  const searchableRowContent =
    `${row.original.title} ${getEventDescriptionText(row.original.description)} ${location}`.toLowerCase()
  const searchTerm = (filterValue ?? '').toLowerCase()
  return searchableRowContent.includes(searchTerm)
}

export const formatFilterFn: FilterFn<Event> = (row, columnId, filterValue: number[]) => {
  if (!filterValue?.length) return true
  const format = row.getValue(columnId) as number
  return filterValue.includes(format)
}

// ============================================================================
// Column Definitions
// ============================================================================

export const columns: ColumnDef<Event>[] = [
  // ID Column - minimal width
  {
    accessorKey: 'id',
    header: 'ID',
    size: 55,
    minSize: 55,
    maxSize: 55,
    enableSorting: true,
    cell: ({ row }) => <span className='font-mono text-xs text-muted-foreground tabular-nums'>#{row.original.id}</span>
  },

  // Image Column
  {
    accessorKey: 'imageUrl',
    header: 'Image',
    size: 52,
    minSize: 52,
    maxSize: 52,
    enableSorting: false,
    cell: ({ row }) => {
      const imageUrl = row.original.imageUrl
      return (
        <div className='flex items-center justify-center'>
          {imageUrl ? (
            <img
              src={imageUrl}
              alt={`Cover for ${row.original.title}`}
              className='h-9 w-9 rounded-lg object-cover bg-muted ring-1 ring-border/50'
              loading='lazy'
            />
          ) : (
            <div className='h-9 w-9 rounded-lg bg-muted/50 flex items-center justify-center ring-1 ring-border/30'>
              <ImageIcon className='h-4 w-4 text-muted-foreground/40' />
            </div>
          )}
        </div>
      )
    }
  },

  // Title & Description Column - primary focus
  {
    accessorKey: 'title',
    header: 'Event',
    size: 240,
    minSize: 180,
    enableSorting: true,
    filterFn: multiColumnFilterFn,
    cell: ({ row }) => {
      const description = getEventDescriptionText(row.original.description)
      return (
        <div className='flex flex-col gap-0.5 min-w-0 py-1'>
          <span className='font-medium text-sm truncate' title={row.original.title}>
            {row.original.title}
          </span>
          {description && <span className='text-xs text-muted-foreground truncate max-w-[200px]'>{description}</span>}
          <EventBadges event={row.original} showRequests className='mt-1' />
        </div>
      )
    }
  },

  // Organization Column - show name if available
  {
    accessorKey: 'organizationId',
    header: 'Organization',
    size: 120,
    minSize: 100,
    enableSorting: false,
    cell: ({ row }) => {
      const org = row.original.organization
      return (
        <div className='flex items-center gap-2 min-w-0'>
          {org?.imageUrl ? (
            <img
              src={org.imageUrl}
              alt={`${org.title} logo`}
              width={24}
              height={24}
              className='h-6 w-6 shrink-0 rounded-full object-cover bg-muted ring-1 ring-border/30'
              loading='lazy'
            />
          ) : (
            <div className='h-6 w-6 shrink-0 rounded-full bg-gradient-to-br from-primary/20 to-primary/10 flex items-center justify-center text-xs font-medium text-primary'>
              {org?.title?.[0]?.toUpperCase() || '#'}
            </div>
          )}
          <span className='text-sm truncate' title={org?.title}>
            {org?.title || `Org #${row.original.organizationId}`}
          </span>
        </div>
      )
    }
  },

  // Category Column
  {
    accessorKey: 'category',
    header: 'Category',
    size: 100,
    minSize: 80,
    enableSorting: false,
    cell: ({ row }) => {
      const category = row.original.category
      if (!category) {
        return <span className='text-xs text-muted-foreground/60 italic'>—</span>
      }
      return (
        <div className='flex items-center gap-2 min-w-0'>
          <div className='h-3 w-3 shrink-0 rounded-full' style={{ backgroundColor: category.color || '#6B7280' }} />
          <span className='text-sm truncate' title={category.name}>
            {category.name}
          </span>
        </div>
      )
    }
  },

  // Location Column
  {
    accessorKey: 'structuredLocation',
    header: 'Location',
    size: 110,
    minSize: 90,
    enableSorting: false,
    cell: ({ row }) => {
      const location = formatStructuredLocation(row.original.structuredLocation)
      if (!location || location === 'TBD') {
        return <span className='text-xs text-muted-foreground/60 italic'>TBD</span>
      }
      return (
        <div className='flex items-center gap-1.5 min-w-0'>
          <MapPinIcon className='h-3.5 w-3.5 text-muted-foreground/70 flex-shrink-0' />
          <span className='text-sm truncate' title={location}>
            {location}
          </span>
        </div>
      )
    }
  },

  // Date Column - compact with relative time
  {
    accessorKey: 'startTime',
    header: 'Date',
    size: 120,
    minSize: 100,
    enableSorting: true,
    cell: ({ row }) => {
      const startTime = row.original.startTime
      if (!startTime) return <span className='text-muted-foreground'>-</span>

      try {
        const date = new Date(startTime)
        const relative = getRelativeTime(startTime)

        // Format: "Jan 15" or "Jan 15, 2024" if different year
        const now = new Date()
        const sameYear = date.getFullYear() === now.getFullYear()
        const dateStr = date.toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
          ...(sameYear ? {} : { year: 'numeric' })
        })

        return (
          <div className='flex flex-col gap-0.5'>
            <div className='flex items-center gap-1.5'>
              <CalendarDaysIcon className='h-3.5 w-3.5 text-muted-foreground/60 flex-shrink-0' />
              <span className='text-sm font-medium tabular-nums'>{dateStr}</span>
            </div>
            {relative.text && (
              <span
                className={cn(
                  'text-xs font-medium pl-5',
                  relative.isUpcoming && 'text-status-upcoming',
                  relative.isPast && 'text-muted-foreground/60'
                )}
              >
                {relative.text}
              </span>
            )}
          </div>
        )
      } catch {
        return <span className='text-muted-foreground'>-</span>
      }
    }
  },

  // Status Column - event lifecycle status
  {
    id: 'status',
    header: 'Status',
    size: 120,
    minSize: 110,
    maxSize: 140,
    enableSorting: false,
    cell: ({ row }) => {
      const status = getEventStatus(row.original)
      return <EventStatusBadge status={status} />
    }
  },

  // Format Column
  {
    accessorKey: 'format',
    header: 'Format',
    size: 85,
    minSize: 85,
    maxSize: 85,
    enableSorting: false,
    filterFn: formatFilterFn,
    cell: ({ row }) => {
      return <FormatBadge format={row.original.format} />
    }
  }
]
