import type { Event } from '@studyverse/proto'
import type { Row } from '@tanstack/react-table'
import { Calendar, ImageIcon, MapPin, Users } from 'lucide-react'
import { parsePlateJson, plateJsonToText } from '@/shared/lib/plate-utils'
import { formatDate } from '@/shared/lib/utils'
import { Card, CardContent, CardHeader } from '@/shared/ui/components/card'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { EventStatusBadge } from '@/shared/ui/components/event-status-badge'
import { getEventStatus } from '../utils/event-status'
import { formatStructuredLocation } from '../utils/format-location'
import { EventBadges } from './event-badges'
import { FormatBadge } from './format-badge'

interface EventCardProps {
  row: Row<Event>
}

/**
 * Mobile-friendly card view for an event row
 */
export function EventCard({ row }: EventCardProps) {
  const event = row.original
  const eventFormat = event.format

  return (
    <Card className='relative'>
      <CardHeader className='flex flex-row items-start gap-3 pb-2'>
        {/* Selection checkbox */}
        <Checkbox
          aria-label='Select event'
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          className='mt-1'
        />

        {/* Event image */}
        {event.imageUrl ? (
          <img src={event.imageUrl} alt={event.title} className='h-16 w-16 rounded-md object-cover bg-muted' />
        ) : (
          <div className='h-16 w-16 rounded-md bg-muted flex items-center justify-center text-muted-foreground'>
            <ImageIcon className='h-6 w-6 opacity-40' />
          </div>
        )}

        {/* Title and description */}
        <div className='flex-1 min-w-0'>
          <h3 className='font-medium text-sm leading-tight line-clamp-2'>{event.title}</h3>
          {event.description && (
            <p className='text-xs text-muted-foreground line-clamp-2 mt-1'>
              {plateJsonToText(parsePlateJson(event.description))}
            </p>
          )}
        </div>

        {/* Format badge */}
        <FormatBadge format={eventFormat} />
      </CardHeader>

      <CardContent className='pt-0'>
        <div className='mb-2'>
          <EventStatusBadge status={getEventStatus(event)} />
          <EventBadges event={event} showRequests className='mt-1.5' />
        </div>
        <div className='flex flex-wrap gap-x-4 gap-y-2 text-xs text-muted-foreground'>
          {/* Date & Time */}
          <div className='flex items-center gap-1'>
            <Calendar className='h-3.5 w-3.5' />
            <span>{formatDate(event.startTime, true)}</span>
          </div>

          {/* Location */}
          {event.structuredLocation && (
            <div className='flex items-center gap-1'>
              <MapPin className='h-3.5 w-3.5' />
              <span className='line-clamp-1'>{formatStructuredLocation(event.structuredLocation)}</span>
            </div>
          )}

          {/* Organization */}
          <div className='flex items-center gap-1'>
            <Users className='h-3.5 w-3.5' />
            <span>Org #{event.organizationId}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

interface EventCardListProps {
  rows: Row<Event>[]
}

/**
 * Mobile card list for events - shows instead of table on small screens
 */
export function EventCardList({ rows }: EventCardListProps) {
  if (rows.length === 0) {
    return <div className='text-center py-8 text-muted-foreground'>No events found.</div>
  }

  return (
    <div className='grid grid-cols-1 gap-3'>
      {rows.map((row) => (
        <EventCard key={row.id} row={row} />
      ))}
    </div>
  )
}

/**
 * Simple card for displaying an event without Row wrapper
 * Used for mobile views where table row features aren't needed
 */
interface SimpleEventCardProps {
  event: Event
  id?: string
}

export function SimpleEventCard({ event }: SimpleEventCardProps) {
  const eventFormat = event.format

  return (
    <Card className='relative'>
      <CardHeader className='flex flex-row items-start gap-3 pb-2'>
        {/* Event image */}
        {event.imageUrl ? (
          <img src={event.imageUrl} alt={event.title} className='h-16 w-16 rounded-md object-cover bg-muted' />
        ) : (
          <div className='h-16 w-16 rounded-md bg-muted flex items-center justify-center text-muted-foreground'>
            <ImageIcon className='h-6 w-6 opacity-40' />
          </div>
        )}

        {/* Title and description */}
        <div className='flex-1 min-w-0'>
          <h3 className='font-medium text-sm leading-tight line-clamp-2'>{event.title}</h3>
          {event.description && (
            <p className='text-xs text-muted-foreground line-clamp-2 mt-1'>
              {plateJsonToText(parsePlateJson(event.description))}
            </p>
          )}
        </div>

        {/* Format badge */}
        <FormatBadge format={eventFormat} />
      </CardHeader>

      <CardContent className='pt-0'>
        <div className='mb-2'>
          <EventStatusBadge status={getEventStatus(event)} />
          <EventBadges event={event} showRequests className='mt-1.5' />
        </div>
        <div className='flex flex-wrap gap-x-4 gap-y-2 text-xs text-muted-foreground'>
          {/* Date & Time */}
          <div className='flex items-center gap-1'>
            <Calendar className='h-3.5 w-3.5' />
            <span>{formatDate(event.startTime, true)}</span>
          </div>

          {/* Location */}
          {event.structuredLocation && (
            <div className='flex items-center gap-1'>
              <MapPin className='h-3.5 w-3.5' />
              <span className='line-clamp-1'>{formatStructuredLocation(event.structuredLocation)}</span>
            </div>
          )}

          {/* Organization */}
          <div className='flex items-center gap-1'>
            <Users className='h-3.5 w-3.5' />
            <span>Org #{event.organizationId}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

interface SimpleEventCardListProps {
  events: Event[]
}

/**
 * Simple mobile card list for events - no row selection needed
 */
export function SimpleEventCardList({ events }: SimpleEventCardListProps) {
  if (events.length === 0) {
    return <div className='text-center py-8 text-muted-foreground'>No events found.</div>
  }

  return (
    <div className='grid grid-cols-1 gap-3'>
      {events.map((event) => (
        <SimpleEventCard key={event.id} event={event} />
      ))}
    </div>
  )
}
