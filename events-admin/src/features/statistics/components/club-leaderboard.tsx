'use client'

import type { GetOrganizationTrendsResponse } from '@studyverse/proto'
import { Calendar, TrendingDown, TrendingUp, Trophy, Users } from 'lucide-react'
import { useMemo } from 'react'
import { useOrganizationTrends, useTopPerformingClubs } from '@/features/statistics'
import { useSchrodingersModal } from '@/shared/hooks/use-schrodingers-modal'
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { SchrodingersModal } from '@/shared/ui/components/schrodingers-modal'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { calculateTrend, Sparkline } from '@/shared/ui/components/sparkline'

interface OrganizationLeaderboardProps {
  limit?: number
  days?: number
}

interface TrendData {
  data: { value: number }[]
  trend: { percentage: number; direction: 'up' | 'down' | 'neutral' }
  color: string
}

function buildTrendsMap(trendsData: GetOrganizationTrendsResponse | undefined) {
  const map = new Map<number, TrendData>()
  if (!trendsData?.trends) return map
  for (const t of trendsData.trends) {
    const data = t.points.map((p) => ({ value: p.value }))
    const trend = calculateTrend(data)
    const color =
      trend.direction === 'up'
        ? 'oklch(0.696 0.149 162.48)'
        : trend.direction === 'down'
          ? 'oklch(0.577 0.245 27.325)'
          : 'var(--muted-foreground)'
    map.set(t.organizationId, { data, trend, color })
  }
  return map
}

export function ClubLeaderboard({ limit = 5, days = 90 }: OrganizationLeaderboardProps) {
  const modal = useSchrodingersModal()
  const { data, isLoading } = useTopPerformingClubs({ limit, days })

  const organizationIds = useMemo(
    () => data?.clubs?.map((club: { organizationId: number }) => club.organizationId) ?? [],
    [data?.clubs]
  )

  const { data: trendsData } = useOrganizationTrends({ organizationIds, days })

  const trendsMap = useMemo(() => buildTrendsMap(trendsData), [trendsData])

  const organizations = data?.clubs || []
  type OrganizationSummary = typeof organizations extends Array<infer T> ? T : never

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className='flex items-center gap-2'>
            <Trophy className='size-5' />
            Top Performing Organizations
          </CardTitle>
          <CardDescription>Most active organizations in the last {days} days</CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          {['leaderboard-skeleton-1', 'leaderboard-skeleton-2', 'leaderboard-skeleton-3'].map((key) => (
            <div key={key} className='flex items-center gap-4'>
              <Skeleton className='size-12 rounded-full' />
              <div className='flex-1 space-y-2'>
                <Skeleton className='h-4 w-32' />
                <Skeleton className='h-3 w-48' />
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    )
  }

  const topOrganizations = organizations.slice(0, limit)

  return (
    <>
      <Card className='h-full'>
        <CardHeader>
          <div className='flex items-center justify-between'>
            <div>
              <CardTitle className='flex items-center gap-2'>
                <Trophy className='size-5' />
                Top Performing Organizations
              </CardTitle>
              <CardDescription>Most active organizations in the last {days} days</CardDescription>
            </div>
            <Button variant='outline' size='sm' onClick={modal.open}>
              See More
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {topOrganizations.length === 0 ? (
            <div className='text-muted-foreground flex h-32 items-center justify-center text-sm'>
              No organization activity in this period
            </div>
          ) : (
            <div className='space-y-4'>
              {topOrganizations.map((org: OrganizationSummary, index: number) => (
                <div key={org.organizationId} className='flex items-center gap-4'>
                  <div className='flex size-10 shrink-0 items-center justify-center'>
                    {index < 3 ? (
                      <div
                        className={`flex size-8 items-center justify-center rounded-full font-bold ${
                          index === 0
                            ? 'bg-yellow-500/20 text-yellow-600 dark:text-yellow-400'
                            : index === 1
                              ? 'bg-gray-400/20 text-gray-600 dark:text-gray-400'
                              : 'bg-orange-500/20 text-orange-600 dark:text-orange-400'
                        }`}
                      >
                        {index + 1}
                      </div>
                    ) : (
                      <span className='text-muted-foreground text-sm font-medium'>{index + 1}</span>
                    )}
                  </div>
                  <Avatar className='size-12'>
                    <AvatarImage src={org.organizationImage || undefined} alt={org.organizationTitle} />
                    <AvatarFallback>{org.organizationTitle.slice(0, 2).toUpperCase()}</AvatarFallback>
                  </Avatar>
                  <div className='flex-1 space-y-1'>
                    <div className='font-medium'>{org.organizationTitle}</div>
                    <div className='flex gap-3 text-xs text-muted-foreground'>
                      <span className='flex items-center gap-1'>
                        <Calendar className='size-3' />
                        {org.totalEvents} events
                      </span>
                      <span className='flex items-center gap-1'>
                        <Users className='size-3' />
                        {org.totalRegistrations} registrations
                      </span>
                    </div>
                  </div>
                  {(() => {
                    const trendInfo = trendsMap.get(org.organizationId)
                    return trendInfo ? (
                      <Sparkline data={trendInfo.data} color={trendInfo.color} height={24} className='w-16' />
                    ) : null
                  })()}
                  {(() => {
                    const trendInfo = trendsMap.get(org.organizationId)
                    const isUp = trendInfo?.trend.direction === 'up'
                    const isDown = trendInfo?.trend.direction === 'down'
                    return (
                      <Badge
                        variant='secondary'
                        className={`ml-auto ${isUp ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : isDown ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : ''}`}
                      >
                        {isDown ? <TrendingDown className='mr-1 size-3' /> : <TrendingUp className='mr-1 size-3' />}
                        {org.averageAttendanceRate.toFixed(0)}%
                      </Badge>
                    )
                  })()}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <SchrodingersModal
        {...modal.props}
        title='Top Performing Organizations'
        description={`All organizations ranked by activity in the last ${days} days`}
        className='sm:max-w-3xl'
      >
        <div className='space-y-4 py-4'>
          {organizations.length === 0 ? (
            <div className='text-muted-foreground flex h-32 items-center justify-center text-sm'>
              No organization activity in this period
            </div>
          ) : (
            organizations.map((org: OrganizationSummary, index: number) => (
              <div key={org.organizationId} className='flex items-center gap-4 rounded-lg border p-4'>
                <div className='flex size-12 shrink-0 items-center justify-center'>
                  {index < 3 ? (
                    <div
                      className={`flex size-10 items-center justify-center rounded-full font-bold text-lg ${
                        index === 0
                          ? 'bg-yellow-500/20 text-yellow-600 dark:text-yellow-400'
                          : index === 1
                            ? 'bg-gray-400/20 text-gray-600 dark:text-gray-400'
                            : 'bg-orange-500/20 text-orange-600 dark:text-orange-400'
                      }`}
                    >
                      {index + 1}
                    </div>
                  ) : (
                    <span className='text-muted-foreground text-lg font-semibold'>{index + 1}</span>
                  )}
                </div>
                <Avatar className='size-14'>
                  <AvatarImage src={org.organizationImage || undefined} alt={org.organizationTitle} />
                  <AvatarFallback>{org.organizationTitle.slice(0, 2).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div className='flex-1 space-y-2'>
                  <div className='text-lg font-semibold'>{org.organizationTitle}</div>
                  <div className='grid grid-cols-3 gap-4 text-sm'>
                    <div>
                      <div className='text-muted-foreground text-xs'>Events</div>
                      <div className='font-semibold'>{org.totalEvents}</div>
                    </div>
                    <div>
                      <div className='text-muted-foreground text-xs'>Registrations</div>
                      <div className='font-semibold'>{org.totalRegistrations}</div>
                    </div>
                    <div>
                      <div className='text-muted-foreground text-xs'>Attendees</div>
                      <div className='font-semibold'>{org.totalAttendees}</div>
                    </div>
                  </div>
                </div>
                <div className='flex flex-col items-end gap-2'>
                  {(() => {
                    const trendInfo = trendsMap.get(org.organizationId)
                    return trendInfo ? (
                      <Sparkline data={trendInfo.data} color={trendInfo.color} height={28} className='w-20' />
                    ) : null
                  })()}
                  {(() => {
                    const trendInfo = trendsMap.get(org.organizationId)
                    const isUp = trendInfo?.trend.direction === 'up'
                    const isDown = trendInfo?.trend.direction === 'down'
                    return (
                      <Badge
                        variant='secondary'
                        className={
                          isUp
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                            : isDown
                              ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                              : ''
                        }
                      >
                        {isDown ? <TrendingDown className='mr-1 size-3' /> : <TrendingUp className='mr-1 size-3' />}
                        {org.averageAttendanceRate.toFixed(1)}%
                      </Badge>
                    )
                  })()}
                </div>
              </div>
            ))
          )}
        </div>
      </SchrodingersModal>
    </>
  )
}
