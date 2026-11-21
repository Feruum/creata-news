import { createFileRoute } from '@tanstack/react-router'
import { Suspense, useState } from 'react'
import { ActivityWidget } from '@/features/activity'
import {
  ChartPieInteractive,
  ChartRadialEngagement,
  ClubLeaderboard,
  type EngagementLevel,
  EventActivityHeatmap,
  StatisticsAccess
} from '@/features/statistics'
import { ChartAreaInteractive } from '@/shared/components/chart-area-interactive'
import { EventAnalytics } from '@/shared/components/event-analytics'
import { SectionCards } from '@/shared/components/section-cards'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Card, CardContent, CardHeader } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'

export const Route = createFileRoute('/_authenticated/')({
  component: () => (
    <Suspense fallback={<DashboardLoading />}>
      <Dashboard />
    </Suspense>
  )
})

const kpiSkeletonIds = ['kpi-1', 'kpi-2', 'kpi-3', 'kpi-4']
const dashboardRowIds = ['row-1', 'row-2', 'row-3', 'row-4', 'row-5']

function DashboardLoading() {
  return (
    <div className='flex flex-col gap-6'>
      {/* KPI Cards */}
      <div className='grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4'>
        {kpiSkeletonIds.map((cardId) => (
          <Card key={cardId}>
            <CardHeader>
              <Skeleton className='h-4 w-24' />
              <Skeleton className='h-8 w-16 mt-2' />
            </CardHeader>
          </Card>
        ))}
      </div>

      {/* Charts Row */}
      <div className='grid grid-cols-1 gap-6 xl:grid-cols-3'>
        <Card className='xl:col-span-2'>
          <CardHeader>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='h-4 w-48 mt-1' />
          </CardHeader>
          <CardContent>
            <Skeleton className='h-[280px] w-full' />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className='h-5 w-28' />
            <Skeleton className='h-4 w-40 mt-1' />
          </CardHeader>
          <CardContent>
            <Skeleton className='h-[280px] w-full' />
          </CardContent>
        </Card>
      </div>

      {/* Analytics Row */}
      <div className='grid grid-cols-1 gap-6 xl:grid-cols-5'>
        <Card className='xl:col-span-2'>
          <CardHeader>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='h-4 w-44 mt-1' />
          </CardHeader>
          <CardContent>
            <Skeleton className='h-[180px] w-full' />
          </CardContent>
        </Card>
        <Card className='xl:col-span-3'>
          <CardHeader>
            <Skeleton className='h-5 w-48' />
            <Skeleton className='h-4 w-52 mt-1' />
          </CardHeader>
          <CardContent className='space-y-3'>
            {dashboardRowIds.map((rowId) => (
              <Skeleton key={rowId} className='h-10 w-full' />
            ))}
          </CardContent>
        </Card>
      </div>

      {/* Heatmap */}
      <Card>
        <CardHeader>
          <Skeleton className='h-5 w-32' />
          <Skeleton className='h-4 w-40 mt-1' />
        </CardHeader>
        <CardContent>
          <Skeleton className='h-[100px] w-full' />
        </CardContent>
      </Card>

      {/* Table */}
      <Card>
        <CardHeader>
          <Skeleton className='h-5 w-36' />
          <Skeleton className='h-4 w-64 mt-1' />
        </CardHeader>
        <CardContent className='space-y-2'>
          {dashboardRowIds.map((rowId) => (
            <Skeleton key={rowId} className='h-12 w-full' />
          ))}
        </CardContent>
      </Card>
    </div>
  )
}

function Dashboard() {
  const currentUser = useCurrentUser()
  const currentYear = new Date().getFullYear()
  const currentMonth = new Date().getMonth() + 1
  const [selectedEngagementStage, setSelectedEngagementStage] = useState<EngagementLevel | null>(null)

  if (currentUser.isLoading) return <DashboardLoading />
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
      {/* KPI Cards Section */}
      <SectionCards />

      {/* Charts Row: Area Chart (2/3) + Pie Chart (1/3) */}
      <div className='grid grid-cols-1 gap-6 xl:grid-cols-3'>
        <ChartAreaInteractive />
        <ChartPieInteractive year={currentYear} month={currentMonth} />
      </div>

      {/* Analytics Row: Radial Chart (2/5) + Leaderboard (3/5) */}
      <div className='grid grid-cols-1 gap-6 xl:grid-cols-5'>
        <div className='xl:col-span-2'>
          <ChartRadialEngagement selectedStage={selectedEngagementStage} onStageSelect={setSelectedEngagementStage} />
        </div>
        <div className='xl:col-span-3'>
          <ClubLeaderboard limit={5} days={90} />
        </div>
      </div>

      {/* Event Activity Heatmap + Team Activity Feed */}
      <div className='grid grid-cols-1 gap-6 xl:grid-cols-7'>
        <div className='xl:col-span-5'>
          <EventActivityHeatmap year={currentYear} />
        </div>
        <ActivityWidget className='xl:col-span-2' />
      </div>

      {/* Event Analytics Table */}
      <EventAnalytics />
    </div>
  )
}
