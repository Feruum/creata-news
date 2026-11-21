import { createFileRoute } from '@tanstack/react-router'
import { Suspense } from 'react'

import {
  ChartPieInteractive,
  ChartRadialEngagement,
  ClubLeaderboard,
  EventActivityHeatmap,
  StatisticsAccess,
  SundayOpenAnalytics,
  SundayOpenAudienceStatistics
} from '@/features/statistics'
import { ChartAreaInteractive } from '@/shared/components/chart-area-interactive'
import { EventAnalytics } from '@/shared/components/event-analytics'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Card, CardContent, CardHeader } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'

const analyticsRowIds = ['row-1', 'row-2', 'row-3', 'row-4', 'row-5']
const leaderboardRowIds = ['leaderboard-1', 'leaderboard-2', 'leaderboard-3', 'leaderboard-4', 'leaderboard-5']

export const Route = createFileRoute('/_authenticated/analytics')({
  component: () => (
    <Suspense fallback={<AnalyticsLoading />}>
      <Analytics />
    </Suspense>
  )
})

function AnalyticsLoading() {
  return (
    <div className='flex flex-col gap-6'>
      {/* Page Header */}
      <div>
        <Skeleton className='h-8 w-40' />
        <Skeleton className='h-4 w-80 mt-2' />
      </div>

      <div className='grid gap-6 lg:grid-cols-3'>
        {/* Left Column */}
        <div className='lg:col-span-2 flex flex-col gap-6'>
          {/* Heatmap */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-32' />
              <Skeleton className='h-4 w-44 mt-1' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-[100px] w-full' />
            </CardContent>
          </Card>

          {/* Area Chart */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-28' />
              <Skeleton className='h-4 w-48 mt-1' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-[280px] w-full' />
            </CardContent>
          </Card>

          {/* Table */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-36' />
              <Skeleton className='h-4 w-64 mt-1' />
            </CardHeader>
            <CardContent className='space-y-2'>
              {analyticsRowIds.map((rowId) => (
                <div key={rowId} className='flex items-center justify-between gap-4'>
                  <Skeleton className='h-4 w-40' />
                  <Skeleton className='h-4 w-28' />
                  <Skeleton className='h-4 w-24' />
                </div>
              ))}
            </CardContent>
          </Card>
        </div>

        {/* Right Column */}
        <div className='flex flex-col gap-6'>
          {/* Radial Chart */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-32' />
              <Skeleton className='h-4 w-44 mt-1' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-[180px] w-full' />
            </CardContent>
          </Card>

          {/* Pie Chart */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-28' />
              <Skeleton className='h-4 w-40 mt-1' />
            </CardHeader>
            <CardContent>
              <Skeleton className='h-[220px] w-full' />
            </CardContent>
          </Card>

          {/* Leaderboard */}
          <Card>
            <CardHeader>
              <Skeleton className='h-5 w-48' />
              <Skeleton className='h-4 w-52 mt-1' />
            </CardHeader>
            <CardContent className='space-y-3'>
              {leaderboardRowIds.map((rowId) => (
                <Skeleton key={rowId} className='h-10 w-full' />
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}

function Analytics() {
  const currentUser = useCurrentUser()
  const currentYear = new Date().getFullYear()
  const currentMonth = new Date().getMonth() + 1

  if (currentUser.isLoading) return <AnalyticsLoading />
  if (currentUser.error)
    return (
      <div role='alert'>
        Не удалось проверить доступ.{' '}
        <button type='button' onClick={currentUser.refetch}>
          Повторить
        </button>
      </div>
    )
  if (!currentUser.isAdmin && !currentUser.isStaff) return <StatisticsAccess />

  return (
    <div className='flex flex-col gap-6'>
      <SundayOpenAnalytics />
      <SundayOpenAudienceStatistics />

      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Event Analytics</h1>
        <p className='text-sm text-muted-foreground mt-1'>
          Analyze event performance, identify opportunities, and track organization activity
        </p>
      </div>

      <div className='grid gap-6 lg:grid-cols-3'>
        <div className='lg:col-span-2 flex flex-col gap-6'>
          <EventActivityHeatmap year={currentYear} />
          <ChartAreaInteractive />
          <EventAnalytics />
        </div>

        <div className='flex flex-col gap-6'>
          <ChartRadialEngagement />
          <ChartPieInteractive year={currentYear} month={currentMonth} />
          <ClubLeaderboard limit={5} days={90} />
        </div>
      </div>
    </div>
  )
}
