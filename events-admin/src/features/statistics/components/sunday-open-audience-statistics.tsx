import { Code, ConnectError } from '@connectrpc/connect'
import type { SundayOpenAudienceBucket } from '@studyverse/proto'
import { UsersRound } from 'lucide-react'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Button } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { useSundayOpenAudienceStatistics } from '../api'

const roleLabels = {
  school_applicant: 'Школьник / абитуриент',
  parent: 'Родитель',
  student: 'Студент',
  educator: 'Преподаватель / сотрудник',
  alumni: 'Выпускник',
  other: 'Другое'
}
const interestLabels = {
  it_ai: 'IT и AI',
  design: 'Дизайн',
  business: 'Бизнес',
  science_engineering: 'Наука и инженерия',
  languages: 'Языки',
  creativity: 'Творчество',
  sports_games: 'Спорт и игры',
  socializing: 'Общение'
}
const goalLabels = {
  try_new: 'Попробовать новое',
  practical_skills: 'Получить практические навыки',
  meet_people: 'Познакомиться с людьми',
  discover_aitu: 'Узнать об AITU',
  choose_career: 'Определиться с профессией',
  find_club: 'Найти клуб'
}
const formatLabels = {
  workshops: 'Воркшопы и мастер-классы',
  talks: 'Выступления и лекции',
  tours: 'Экскурсии',
  club_meetups: 'Встречи с клубами',
  competitions: 'Соревнования и хакатоны'
}
const sourceLabels = {
  instagram: 'Instagram',
  telegram: 'Telegram',
  tiktok: 'TikTok',
  youtube: 'YouTube',
  friends: 'Друзья / знакомые',
  school_university: 'Школа / университет',
  search: 'Поиск в интернете',
  other: 'Другое'
}

export function SundayOpenAudienceStatistics() {
  const currentUser = useCurrentUser()
  if (currentUser.isLoading || (!currentUser.isAdmin && !currentUser.isStaff)) return null
  return <AudienceStatisticsContent />
}

function AudienceStatisticsContent() {
  const { data, error, isPending, isFetching, refetch } = useSundayOpenAudienceStatistics()
  const denied = ConnectError.from(error).code === Code.PermissionDenied
  return (
    <section aria-labelledby='sunday-open-audience-title' className='flex min-w-0 flex-col gap-4'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <h2 id='sunday-open-audience-title' className='flex items-center gap-2 text-xl font-semibold tracking-tight'>
            <UsersRound aria-hidden='true' className='h-5 w-5 shrink-0' />
            Аудитория Sunday Open
          </h2>
          <p className='mt-1 text-sm text-muted-foreground'>Ответы участников за всё время</p>
        </div>
        <Button variant='outline' size='sm' disabled={isFetching} onClick={() => void refetch()}>
          Обновить
        </Button>
      </div>
      {isPending && !error ? (
        <div role='status' aria-label='Загружаем статистику аудитории' className='grid gap-4 sm:grid-cols-2'>
          <Skeleton className='h-32 rounded-xl' />
          <Skeleton className='h-32 rounded-xl' />
        </div>
      ) : error ? (
        <Card>
          <CardContent className='flex flex-wrap items-center justify-between gap-3 pt-6'>
            <p role='alert' className='text-sm text-destructive'>
              {denied ? 'Нет доступа к статистике аудитории' : 'Не удалось загрузить статистику'}
            </p>
            {!denied && (
              <Button variant='outline' onClick={() => void refetch()}>
                Повторить
              </Button>
            )}
          </CardContent>
        </Card>
      ) : data ? (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Заполненные анкеты</CardTitle>
              <CardDescription>
                {data.completedAccounts} из {data.totalAccounts} аккаунтов
              </CardDescription>
            </CardHeader>
            <CardContent>
              <p className='text-3xl font-semibold tabular-nums'>
                {data.totalAccounts > 0 ? Math.round((data.completedAccounts / data.totalAccounts) * 100) : 0}%
              </p>
              <p className='mt-2 text-xs text-muted-foreground'>Каждый аккаунт учитывается один раз.</p>
            </CardContent>
          </Card>
          {data.completedAccounts === 0 ? (
            <Card>
              <CardContent className='py-8 text-sm text-muted-foreground'>Пока нет заполненных анкет</CardContent>
            </Card>
          ) : (
            <>
              <p id='sunday-open-audience-hint' className='text-xs text-muted-foreground'>
                Проценты — от числа заполненных анкет. В интересах, целях и форматах можно выбрать несколько ответов,
                поэтому сумма может превышать 100%.
              </p>
              <div className='grid min-w-0 gap-4 xl:grid-cols-2'>
                <Distribution
                  id='roles'
                  title='Кто приходит'
                  labels={roleLabels}
                  buckets={data.roles}
                  total={data.completedAccounts}
                />
                <Distribution
                  id='interests'
                  title='Что интересно участникам'
                  labels={interestLabels}
                  buckets={data.interests}
                  total={data.completedAccounts}
                />
                <Distribution
                  id='goals'
                  title='Зачем приходят'
                  labels={goalLabels}
                  buckets={data.goals}
                  total={data.completedAccounts}
                />
                <Distribution
                  id='formats'
                  title='Любимые форматы'
                  labels={formatLabels}
                  buckets={data.formats}
                  total={data.completedAccounts}
                />
                <Distribution
                  id='sources'
                  title='Как узнают о нас'
                  labels={sourceLabels}
                  buckets={data.sources}
                  total={data.completedAccounts}
                />
              </div>
            </>
          )}
        </>
      ) : null}
    </section>
  )
}

function Distribution({
  id,
  title,
  labels,
  buckets,
  total
}: {
  id: string
  title: string
  labels: Record<string, string>
  buckets: SundayOpenAudienceBucket[]
  total: number
}) {
  const counts = new Map(buckets.map((bucket) => [bucket.key, bucket.count]))
  const headingId = `sunday-open-audience-${id}`
  return (
    <section aria-labelledby={headingId} aria-describedby='sunday-open-audience-hint' className='min-w-0'>
      <Card className='h-full'>
        <CardHeader>
          <CardTitle id={headingId}>{title}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          {Object.entries(labels).map(([key, label]) => {
            const count = counts.get(key) ?? 0
            const percentage = total > 0 ? Math.round((count / total) * 100) : 0
            return (
              <div key={key} className='space-y-1.5'>
                <div className='flex items-start justify-between gap-3 text-sm'>
                  <span className='min-w-0 break-words'>{label}</span>
                  <span className='shrink-0 tabular-nums text-muted-foreground'>
                    {count} · {percentage}%
                  </span>
                </div>
                <meter
                  min={0}
                  max={100}
                  value={percentage}
                  aria-label={label}
                  aria-valuenow={percentage}
                  className='block h-2 w-full appearance-none [&::-moz-meter-bar]:bg-primary/75 [&::-webkit-meter-bar]:rounded-full [&::-webkit-meter-bar]:border-0 [&::-webkit-meter-bar]:bg-muted [&::-webkit-meter-optimum-value]:bg-primary/75'
                >
                  {percentage}%
                </meter>
              </div>
            )
          })}
        </CardContent>
      </Card>
    </section>
  )
}
