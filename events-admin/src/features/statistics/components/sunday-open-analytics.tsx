import { Code, ConnectError } from '@connectrpc/connect'
import { Button } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { useSundayOpenAnalytics } from '../api'

export function SundayOpenAnalytics() {
  const { data, error, isPending, isFetching, refetch } = useSundayOpenAnalytics()
  const denied = error && ConnectError.from(error).code === Code.PermissionDenied
  const rate =
    data && data.totalRegistrations > 0 ? Math.round((data.totalAttendees / data.totalRegistrations) * 100) : 0
  const maxWeek = Math.max(1, ...(data?.weeks.map((week) => week.registrations) ?? []))
  const maxEvent = Math.max(1, ...(data?.topEvents.map((event) => event.registrations) ?? []))

  return (
    <section aria-labelledby='sunday-open-analytics-title' className='flex min-w-0 flex-col gap-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h1 id='sunday-open-analytics-title' className='text-2xl font-semibold'>
            Sunday Open · Analytics
          </h1>
          <p className='mt-1 text-sm text-muted-foreground'>События, регистрации и посещения за всё время</p>
        </div>
        <Button variant='outline' size='sm' disabled={isFetching} onClick={() => void refetch()}>
          Обновить
        </Button>
      </div>
      {error ? (
        <Card>
          <CardContent className='space-y-3 pt-6'>
            <p role='alert'>
              {denied ? 'Нет доступа к аналитике Sunday Open' : 'Не удалось загрузить аналитику Sunday Open'}
            </p>
            {!denied && (
              <Button variant='outline' onClick={() => void refetch()}>
                Повторить
              </Button>
            )}
          </CardContent>
        </Card>
      ) : isPending ? (
        <div
          role='status'
          aria-label='Загружаем аналитику Sunday Open'
          className='grid gap-4 sm:grid-cols-2 xl:grid-cols-4'
        >
          {['events', 'registrations', 'attendance', 'external'].map((key) => (
            <Skeleton key={key} className='h-32 rounded-xl' />
          ))}
        </div>
      ) : data ? (
        <>
          <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-4'>
            <Metric label='События' value={data.totalEvents} detail='Опубликованные и архивные' />
            <Metric
              label='Регистрации'
              value={data.totalRegistrations}
              detail={`EMS: ${data.emsRegistrations} · внешние: ${data.externalRegistrations}`}
            />
            <Metric label='Посещения' value={data.totalAttendees} detail={`${rate}% активных регистраций`} />
            <Metric
              label='Внешние регистрации'
              value={data.externalRegistrations}
              detail='Канал регистрации через сайт'
            />
          </div>
          {data.totalEvents === 0 && <p className='text-sm text-muted-foreground'>Пока нет событий Sunday Open</p>}
          <div className='grid min-w-0 gap-4 xl:grid-cols-5'>
            <Card className='min-w-0 xl:col-span-3'>
              <CardHeader>
                <CardTitle>Активные регистрации по неделям</CardTitle>
                <CardDescription>Последние семь недель · Asia/Qyzylorda</CardDescription>
              </CardHeader>
              <CardContent>
                {data.totalRegistrations === 0 && (
                  <p className='mb-4 text-sm text-muted-foreground'>Пока нет активных регистраций</p>
                )}
                <ul className='flex h-52 items-end gap-2' aria-label='Регистрации по неделям'>
                  {data.weeks.map((week) => (
                    <li
                      key={week.weekStart}
                      aria-label={`${week.weekStart}: ${week.registrations}`}
                      className='flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-2'
                    >
                      <span className='text-xs tabular-nums'>{week.registrations}</span>
                      <div
                        className='w-full max-w-12 rounded-t bg-primary/75'
                        style={{ height: `${(week.registrations / maxWeek) * 75}%` }}
                      />
                      <span className='text-[10px] text-muted-foreground'>{week.weekStart.slice(5)}</span>
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
            <Card className='min-w-0 xl:col-span-2'>
              <CardHeader>
                <CardTitle>Регистрации по событиям</CardTitle>
                <CardDescription>Четыре события с наибольшим числом регистраций за всё время</CardDescription>
              </CardHeader>
              <CardContent className='space-y-4'>
                {data.topEvents.length === 0 ? (
                  <p className='text-sm text-muted-foreground'>Пока нет событий для сравнения</p>
                ) : (
                  data.topEvents.map((event) => (
                    <div key={event.eventId} className='space-y-2'>
                      <div className='flex justify-between gap-3 text-sm'>
                        <span className='min-w-0 break-words'>{event.title}</span>
                        <span className='tabular-nums'>{event.registrations}</span>
                      </div>
                      <div className='h-2 rounded bg-muted'>
                        <div
                          className='h-full rounded bg-primary/75'
                          style={{ width: `${(event.registrations / maxEvent) * 100}%` }}
                        />
                      </div>
                    </div>
                  ))
                )}
              </CardContent>
            </Card>
          </div>
          <p className='text-xs text-muted-foreground'>
            Учитываются активные записи регистрации. Внешний канал регистрации не определяет принадлежность участника к
            AITU.
          </p>
        </>
      ) : null}
    </section>
  )
}

function Metric({ label, value, detail }: { label: string; value: number; detail: string }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{label}</CardDescription>
      </CardHeader>
      <CardContent>
        <p className='text-2xl font-semibold tabular-nums'>{value}</p>
        <p className='mt-1 text-xs text-muted-foreground'>{detail}</p>
      </CardContent>
    </Card>
  )
}
