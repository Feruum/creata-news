import { format, isSameDay, isWithinInterval, parseISO } from 'date-fns'
import { Calendar, Clock, User } from 'lucide-react'
import { useEffect, useMemo, useRef } from 'react'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import { DroppableArea } from '@/features/calendar/dnd/droppable-area'
import { groupEvents } from '@/features/calendar/helpers'
import type { IEvent } from '@/features/calendar/interfaces'
import { CalendarTimeline } from '@/features/calendar/views/week-and-day-view/calendar-time-line'
import { DayViewMultiDayEventsRow } from '@/features/calendar/views/week-and-day-view/day-view-multi-day-events-row'
import { RenderGroupedEvents } from '@/features/calendar/views/week-and-day-view/render-grouped-events'
import { CreateEventTrigger, EventDetailsModal } from '@/features/events'
import { Card, CardContent } from '@/shared/ui/components/card'
import { DayPicker } from '@/shared/ui/components/day-picker'
import { ScrollArea } from '@/shared/ui/components/scroll-area'

interface IProps {
  singleDayEvents: IEvent[]
  multiDayEvents: IEvent[]
}

export function CalendarDayView({ singleDayEvents, multiDayEvents }: IProps) {
  const { selectedDate, setSelectedDate, users, use24HourFormat } = useCalendar()
  const scrollAreaRef = useRef<HTMLDivElement>(null)

  const hours = Array.from({ length: 24 }, (_, i) => i)

  const allEvents = useMemo(() => [...singleDayEvents, ...multiDayEvents], [singleDayEvents, multiDayEvents])

  const datesWithEvents = useMemo(() => {
    const dates: Date[] = []
    for (const event of allEvents) {
      const eventDate = parseISO(event.startDate)
      if (!dates.some((d) => isSameDay(d, eventDate))) {
        dates.push(eventDate)
      }
    }
    return dates
  }, [allEvents])

  const selectedDayEvents = useMemo(() => {
    return allEvents
      .filter((event) => {
        const eventDate = parseISO(event.startDate)
        return isSameDay(eventDate, selectedDate)
      })
      .sort((a, b) => parseISO(a.startDate).getTime() - parseISO(b.startDate).getTime())
  }, [allEvents, selectedDate])

  const currentlyHappeningCount = useMemo(() => {
    const now = new Date()
    return selectedDayEvents.filter((event) =>
      isWithinInterval(now, {
        start: parseISO(event.startDate),
        end: parseISO(event.endDate)
      })
    ).length
  }, [selectedDayEvents])

  useEffect(() => {
    const handleDragOver = (e: DragEvent) => {
      if (!scrollAreaRef.current) return

      const scrollArea = scrollAreaRef.current
      const rect = scrollArea.getBoundingClientRect()
      const scrollSpeed = 15

      const scrollContainer = scrollArea.querySelector('[data-radix-scroll-area-viewport]') || scrollArea

      if (e.clientY < rect.top + 60) {
        scrollContainer.scrollTop -= scrollSpeed
      }

      if (e.clientY > rect.bottom - 60) {
        scrollContainer.scrollTop += scrollSpeed
      }
    }

    document.addEventListener('dragover', handleDragOver)
    return () => {
      document.removeEventListener('dragover', handleDragOver)
    }
  }, [])

  const dayEvents = singleDayEvents.filter((event) => {
    const eventDate = parseISO(event.startDate)
    return (
      eventDate.getDate() === selectedDate.getDate() &&
      eventDate.getMonth() === selectedDate.getMonth() &&
      eventDate.getFullYear() === selectedDate.getFullYear()
    )
  })

  const groupedEvents = groupEvents(dayEvents)

  return (
    <div className='flex overflow-hidden'>
      <div className='flex min-w-0 flex-1 flex-col overflow-hidden'>
        <div>
          <DayViewMultiDayEventsRow selectedDate={selectedDate} multiDayEvents={multiDayEvents} />

          {/* Day header */}
          <div className='relative z-20 flex border-b border-border'>
            <div className='w-[72px]'></div>
            <span className='flex-1 border-l border-border py-2 text-center text-xs font-medium text-muted-foreground'>
              {format(selectedDate, 'EE')}{' '}
              <span className='font-semibold text-foreground'>{format(selectedDate, 'd')}</span>
            </span>
          </div>
        </div>

        <ScrollArea className='h-[800px]' ref={scrollAreaRef}>
          <div className='flex'>
            {/* Hours column */}
            <div className='relative w-[72px]'>
              {hours.map((hour, index) => (
                <div key={hour} className='relative' style={{ height: '96px' }}>
                  <div className='absolute -top-3 right-2 flex h-6 items-center'>
                    {index !== 0 && (
                      <span className='text-xs text-muted-foreground'>
                        {format(new Date().setHours(hour, 0, 0, 0), use24HourFormat ? 'HH:00' : 'h a')}
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {/* Day grid */}
            <div className='relative flex-1 border-l border-border'>
              <div className='relative'>
                {hours.map((hour, index) => (
                  <div key={hour} className='relative' style={{ height: '96px' }}>
                    {index !== 0 && (
                      <div className='pointer-events-none absolute inset-x-0 top-0 border-b border-border'></div>
                    )}

                    <DroppableArea
                      date={selectedDate}
                      hour={hour}
                      minute={0}
                      className='absolute inset-x-0 top-0 h-[48px]'
                    >
                      <CreateEventTrigger startDate={selectedDate} startTime={{ hour, minute: 0 }}>
                        <div className='absolute inset-0 cursor-pointer transition-colors hover:bg-secondary' />
                      </CreateEventTrigger>
                    </DroppableArea>

                    <div className='pointer-events-none absolute inset-x-0 top-1/2 border-b border-dashed border-border/50'></div>

                    <DroppableArea
                      date={selectedDate}
                      hour={hour}
                      minute={30}
                      className='absolute inset-x-0 bottom-0 h-[48px]'
                    >
                      <CreateEventTrigger startDate={selectedDate} startTime={{ hour, minute: 30 }}>
                        <div className='absolute inset-0 cursor-pointer transition-colors hover:bg-secondary' />
                      </CreateEventTrigger>
                    </DroppableArea>
                  </div>
                ))}

                <RenderGroupedEvents groupedEvents={groupedEvents} day={selectedDate} />
              </div>

              <CalendarTimeline />
            </div>
          </div>
        </ScrollArea>
      </div>

      <div className='hidden w-80 shrink-0 divide-y divide-border border-l border-border md:block'>
        <DayPicker
          className='mx-auto w-fit'
          mode='single'
          selected={selectedDate}
          onSelect={(date) => date && setSelectedDate(date)}
          showWeekNumber
          modifiers={{ hasEvent: datesWithEvents }}
          modifiersClassNames={{ hasEvent: 'has-event-dot' }}
        />

        <div className='flex-1 space-y-3'>
          <div className='flex items-center justify-between px-4 pt-4'>
            <p className='text-sm font-semibold text-foreground'>{format(selectedDate, 'MMMM d')}</p>
            {currentlyHappeningCount > 0 && (
              <span className='flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400'>
                <span className='size-2 rounded-full bg-green-600 animate-pulse' />
                {currentlyHappeningCount} now
              </span>
            )}
          </div>

          {selectedDayEvents.length === 0 ? (
            <p className='p-4 text-center text-sm italic text-muted-foreground'>No events on this day</p>
          ) : (
            <ScrollArea className='h-[422px] px-4'>
              <div className='space-y-3 pb-4'>
                {selectedDayEvents.map((event) => {
                  const user = users.find((user) => user.id === event.user.id)
                  const now = new Date()
                  const isHappening = isWithinInterval(now, {
                    start: parseISO(event.startDate),
                    end: parseISO(event.endDate)
                  })

                  return (
                    <EventDetailsModal key={event.id} eventId={event.id}>
                      <Card
                        className={`cursor-pointer transition-colors hover:bg-muted/50 ${isHappening ? 'ring-1 ring-green-500/50' : ''}`}
                      >
                        <CardContent className='p-3 space-y-2'>
                          <div className='flex items-start justify-between gap-2'>
                            <p className='line-clamp-2 text-sm font-semibold'>{event.title}</p>
                            {isHappening && (
                              <span className='shrink-0 size-2 rounded-full bg-green-600 animate-pulse mt-1.5' />
                            )}
                          </div>

                          {user && (
                            <div className='flex items-center gap-1.5'>
                              <User className='size-3.5 text-muted-foreground' />
                              <span className='text-xs text-muted-foreground'>{user.name}</span>
                            </div>
                          )}

                          <div className='flex items-center gap-3 text-xs text-muted-foreground'>
                            <div className='flex items-center gap-1'>
                              <Calendar className='size-3.5' />
                              <span>{format(new Date(event.startDate), 'MMM d')}</span>
                            </div>
                            <div className='flex items-center gap-1'>
                              <Clock className='size-3.5' />
                              <span>
                                {format(parseISO(event.startDate), use24HourFormat ? 'HH:mm' : 'h:mm a')}
                                {' – '}
                                {format(parseISO(event.endDate), use24HourFormat ? 'HH:mm' : 'h:mm a')}
                              </span>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    </EventDetailsModal>
                  )
                })}
              </div>
            </ScrollArea>
          )}
        </div>
      </div>
    </div>
  )
}
