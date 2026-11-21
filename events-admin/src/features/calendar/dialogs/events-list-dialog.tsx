import { format } from 'date-fns'
import type { ReactElement } from 'react'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import { formatTime } from '@/features/calendar/helpers'
import type { IEvent } from '@/features/calendar/interfaces'
import { dayCellVariants } from '@/features/calendar/views/month-view/day-cell-variants'
import { EventBullet } from '@/features/calendar/views/month-view/event-bullet'
import { EventDetailsModal } from '@/features/events'
import { Modal, ModalContent, ModalHeader, ModalTitle, ModalTrigger } from '@/shared/ui/components/responsive-modal'
import { cn } from '@/shared/ui/lib/utils'

interface EventListDialogProps {
  date: Date
  events: IEvent[]
  maxVisibleEvents?: number
  children?: ReactElement
}

export function EventListDialog({ date, events, maxVisibleEvents = 3, children }: EventListDialogProps) {
  const cellEvents = events
  const hiddenEventsCount = Math.max(cellEvents.length - maxVisibleEvents, 0)
  const { badgeVariant, use24HourFormat } = useCalendar()

  const defaultTrigger = (
    <span className='cursor-pointer'>
      <span className='sm:hidden'>+{hiddenEventsCount}</span>
      <span className='hidden sm:inline py-0.5 px-2 my-1 rounded-xl border'>
        {hiddenEventsCount}
        <span className='mx-1'>more...</span>
      </span>
    </span>
  )

  return (
    <Modal>
      <ModalTrigger render={children || defaultTrigger}></ModalTrigger>
      <ModalContent className='sm:max-w-[425px]'>
        <ModalHeader>
          <ModalTitle className='my-2'>
            <div className='flex items-center gap-2'>
              <EventBullet color={cellEvents[0]?.color} className='' />
              <p className='text-sm font-medium'>Events on {format(date, 'EEEE, MMMM d, yyyy')}</p>
            </div>
          </ModalTitle>
        </ModalHeader>
        <div className='max-h-[60vh] overflow-y-auto space-y-2'>
          {cellEvents.length > 0 ? (
            cellEvents.map((event) => (
              <EventDetailsModal eventId={event.id} key={event.id}>
                <div
                  className={cn('flex items-center gap-2 p-2 border rounded-md hover:bg-muted cursor-pointer', {
                    [dayCellVariants({ color: event.color })]: badgeVariant === 'colored'
                  })}
                >
                  <EventBullet color={event.color} />
                  <div className='flex justify-between items-center w-full'>
                    <p className='text-sm font-medium'>{event.title}</p>
                    <p className='text-xs'>{formatTime(event.startDate, use24HourFormat)}</p>
                  </div>
                </div>
              </EventDetailsModal>
            ))
          ) : (
            <p className='text-sm text-muted-foreground'>No events for this date.</p>
          )}
        </div>
      </ModalContent>
    </Modal>
  )
}
