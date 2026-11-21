import { formatDate } from '@/shared/lib/utils'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import type { ScheduleLesson } from '../types'

const lessonDateOptions: Intl.DateTimeFormatOptions = {
  timeZone: 'Asia/Almaty',
  month: 'short',
  day: 'numeric',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false
}

export function ScheduleLessonList({ events, label }: { events: ScheduleLesson[]; label: string }) {
  if (!events.length) return <p className='text-sm text-muted-foreground'>No lessons.</p>

  return (
    <div className='min-w-0 rounded-lg border'>
      <Table aria-label={label}>
        <TableHeader>
          <TableRow>
            <TableHead scope='col'>Lesson</TableHead>
            <TableHead scope='col'>Start / end (Astana)</TableHead>
            <TableHead scope='col'>Teacher</TableHead>
            <TableHead scope='col'>Type</TableHead>
            <TableHead scope='col'>Location</TableHead>
            <TableHead scope='col'>Format</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {events.map((event) => (
            <TableRow key={event.id}>
              <TableCell className='min-w-40 whitespace-normal font-medium'>{event.title}</TableCell>
              <TableCell className='whitespace-nowrap'>
                <time dateTime={event.start}>{formatDate(event.start, lessonDateOptions)}</time>
                <br />
                <time dateTime={event.end} className='text-muted-foreground'>
                  {formatDate(event.end, lessonDateOptions)}
                </time>
              </TableCell>
              <TableCell className='min-w-32 whitespace-normal'>{event.teacher || '—'}</TableCell>
              <TableCell>{event.lessonType || '—'}</TableCell>
              <TableCell className='min-w-24 whitespace-normal'>{event.location || '—'}</TableCell>
              <TableCell>{event.online ? 'Online' : 'On campus'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
