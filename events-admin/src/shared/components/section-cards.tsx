'use client'

import { useQuery } from '@connectrpc/connect-query'
import { getOverallStatistics } from '@studyverse/proto'
import { TrendingDown, TrendingUp } from 'lucide-react'

import { attendanceAnalyticsQueryOptions } from '@/features/events'
import { Card, CardContent, CardHeader } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { cn } from '@/shared/ui/lib/utils'

function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toLocaleString()
}

interface TrendIndicatorProps {
  value: number
  label: string
  isPositive?: boolean
  showSign?: boolean
}

function TrendIndicator({ value, label, isPositive = true, showSign = true }: TrendIndicatorProps) {
  const Icon = isPositive ? TrendingUp : TrendingDown
  const colorClass = isPositive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'

  const formattedValue = typeof value === 'number' ? value.toFixed(1) : value

  return (
    <span className={cn('inline-flex items-center gap-1 text-sm font-medium', colorClass)}>
      <Icon className='size-3.5' />
      {showSign && value > 0 && '+'}
      {formattedValue} {label}
    </span>
  )
}

interface KPICardProps {
  label: string
  value: string | number
  trend?: {
    value: number
    label: string
    isPositive: boolean
    showSign?: boolean
  }
  comparison?: string
  subtitle?: string
}

function KPICard({ label, value, trend, comparison, subtitle }: KPICardProps) {
  return (
    <Card className='@container/card'>
      <CardHeader className='pb-2'>
        <p className='text-xs font-medium uppercase tracking-wider text-muted-foreground'>{label}</p>
      </CardHeader>
      <CardContent className='pt-0'>
        <div className='flex items-baseline gap-3'>
          <span className='text-4xl font-bold tabular-nums tracking-tight'>{value}</span>
          {trend && (
            <TrendIndicator
              value={trend.value}
              label={trend.label}
              isPositive={trend.isPositive}
              showSign={trend.showSign}
            />
          )}
        </div>
        {(comparison || subtitle) && (
          <div className='mt-2 space-y-0.5'>
            {comparison && <p className='text-sm text-muted-foreground'>{comparison}</p>}
            {subtitle && <p className='text-xs text-muted-foreground/70'>{subtitle}</p>}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function SectionCards() {
  const { data, isLoading } = useQuery(getOverallStatistics, {}, attendanceAnalyticsQueryOptions)

  if (isLoading) {
    return (
      <div className='grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4'>
        {['events', 'registrations', 'attendance', 'organizations'].map((metric) => (
          <Card key={metric} className='@container/card'>
            <CardHeader className='pb-2'>
              <Skeleton className='h-3 w-20' />
            </CardHeader>
            <CardContent className='pt-0'>
              <Skeleton className='h-10 w-24' />
              <Skeleton className='mt-2 h-4 w-32' />
            </CardContent>
          </Card>
        ))}
      </div>
    )
  }

  const stats = data
  const eventsThisMonth = stats?.eventsThisMonth ?? 0
  const registrationsThisMonth = stats?.registrationsThisMonth ?? 0
  const attendanceRate = stats?.averageAttendanceRate ?? 0
  const upcomingEvents = stats?.upcomingEvents ?? 0

  return (
    <div className='grid grid-cols-1 gap-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4'>
      <KPICard
        label='Total Events'
        value={formatNumber(stats?.totalEvents ?? 0)}
        trend={{
          value: eventsThisMonth,
          label: 'this month',
          isPositive: eventsThisMonth > 0
        }}
        comparison={`${upcomingEvents} upcoming`}
        subtitle='Events across all organizations'
      />

      <KPICard
        label='Total Users'
        value={formatNumber(stats?.totalUsers ?? 0)}
        comparison='Platform users'
        subtitle='Total registered accounts'
      />

      <KPICard
        label='Registrations'
        value={formatNumber(stats?.totalRegistrations ?? 0)}
        trend={{
          value: registrationsThisMonth,
          label: 'this month',
          isPositive: registrationsThisMonth > 0
        }}
        comparison='Event sign-ups'
        subtitle='Total registrations all-time'
      />

      <KPICard
        label='Attendance Rate'
        value={`${attendanceRate.toFixed(1)}%`}
        trend={{
          value: attendanceRate,
          label: attendanceRate >= 70 ? 'Good' : 'Avg',
          isPositive: attendanceRate >= 70,
          showSign: false
        }}
        comparison={`${attendanceRate >= 80 ? 'Excellent' : attendanceRate >= 70 ? 'Strong' : 'Moderate'} performance`}
        subtitle='Average across all events'
      />
    </div>
  )
}
