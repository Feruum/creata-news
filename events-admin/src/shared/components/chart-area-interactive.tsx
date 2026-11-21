'use client'

import { useQuery } from '@connectrpc/connect-query'
import { getEventTrends } from '@studyverse/proto'
import { TrendingUp } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Area, Bar, CartesianGrid, ComposedChart, XAxis, YAxis } from 'recharts'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import {
  type ChartConfig,
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent
} from '@/shared/ui/components/chart'
import { ToggleGroup, ToggleGroupItem } from '@/shared/ui/components/toggle-group'

const chartConfig = {
  eventCount: {
    label: 'Events',
    theme: { light: '#4d7c0f', dark: '#a3e635' }
  },
  registrationCount: {
    label: 'Current registrations',
    theme: { light: '#65a30d', dark: '#bef264' }
  }
} satisfies ChartConfig

function formatDate(value: string) {
  return new Date(`${value}T00:00:00Z`).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC'
  })
}

export function ChartAreaInteractive() {
  const [timeRange, setTimeRange] = useState('30')
  const { data, isLoading, isError, refetch } = useQuery(getEventTrends, { days: Number(timeRange) })

  const chartData = useMemo(() => {
    const trends = new Map(data?.trends.map((trend) => [trend.date, trend]))
    const today = new Date()
    return Array.from({ length: Number(timeRange) }, (_, index) => {
      const day = new Date(today)
      day.setUTCDate(day.getUTCDate() - Number(timeRange) + 1 + index)
      const date = day.toISOString().slice(0, 10)
      const trend = trends.get(date)
      return { date, eventCount: trend?.eventCount ?? 0, registrationCount: trend?.registrationCount ?? 0 }
    })
  }, [data, timeRange])

  const totals = chartData.reduce(
    (sum, day) => ({ events: sum.events + day.eventCount, registrations: sum.registrations + day.registrationCount }),
    { events: 0, registrations: 0 }
  )
  const busiest = chartData.reduce<(typeof chartData)[number] | null>(
    (best, day) => (day.registrationCount > (best?.registrationCount ?? 0) ? day : best),
    null
  )
  const hasData = totals.events > 0 || totals.registrations > 0

  return (
    <Card className='@container/card col-span-1 xl:col-span-2'>
      <CardHeader className='flex flex-wrap items-start justify-between gap-3'>
        <div className='space-y-1.5'>
          <CardTitle className='flex items-center gap-2'>
            <TrendingUp className='size-5' />
            Event Activity
          </CardTitle>
          <CardDescription>Events and current registrations, grouped by event date</CardDescription>
        </div>
        <ToggleGroup
          aria-label='Activity period'
          multiple={false}
          value={[timeRange]}
          onValueChange={(value) => {
            if (value[0]) setTimeRange(value[0])
          }}
        >
          {['7', '30', '90'].map((days) => (
            <ToggleGroupItem key={days} value={days} aria-label={`Last ${days} days`} className='text-xs'>
              {days}d
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </CardHeader>
      <CardContent className='space-y-4'>
        {isLoading ? (
          <div className='flex h-[280px] items-center justify-center text-sm text-muted-foreground' role='status'>
            Loading activity…
          </div>
        ) : isError ? (
          <div className='flex h-[280px] flex-col items-center justify-center gap-2 text-sm' role='alert'>
            <p>Could not load event activity.</p>
            <button type='button' className='underline underline-offset-4' onClick={() => void refetch()}>
              Try again
            </button>
          </div>
        ) : (
          <>
            <dl className='grid grid-cols-2 gap-3' aria-label={`Totals for the last ${timeRange} days`}>
              <div className='rounded-xl border bg-muted/30 p-3'>
                <dt className='text-xs text-muted-foreground'>Events in period</dt>
                <dd className='mt-1 text-2xl font-semibold tabular-nums'>{totals.events.toLocaleString()}</dd>
              </div>
              <div className='rounded-xl border bg-muted/30 p-3'>
                <dt className='text-xs text-muted-foreground'>Current registrations</dt>
                <dd className='mt-1 text-2xl font-semibold tabular-nums'>{totals.registrations.toLocaleString()}</dd>
              </div>
            </dl>
            {hasData ? (
              <>
                <ChartContainer config={chartConfig} className='aspect-auto h-[280px] w-full'>
                  <ComposedChart data={chartData} margin={{ left: 0, right: 8, top: 8, bottom: 8 }} accessibilityLayer>
                    <ChartLegend verticalAlign='top' content={<ChartLegendContent className='flex-wrap pb-4' />} />
                    <CartesianGrid vertical={false} strokeDasharray='3 3' />
                    <XAxis
                      dataKey='date'
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      minTickGap={32}
                      tickFormatter={formatDate}
                    />
                    <YAxis tickLine={false} axisLine={false} width={32} allowDecimals={false} />
                    <ChartTooltip
                      content={<ChartTooltipContent labelFormatter={(value) => formatDate(String(value))} />}
                    />
                    <Bar
                      dataKey='eventCount'
                      fill='var(--color-eventCount)'
                      radius={[4, 4, 0, 0]}
                      maxBarSize={24}
                      isAnimationActive={false}
                    />
                    <Area
                      type='linear'
                      dataKey='registrationCount'
                      stroke='var(--color-registrationCount)'
                      fill='var(--color-registrationCount)'
                      fillOpacity={0.12}
                      strokeWidth={2}
                      dot={false}
                      isAnimationActive={false}
                    />
                  </ComposedChart>
                </ChartContainer>
                {busiest && (
                  <p className='text-sm text-muted-foreground'>
                    Most registrations:{' '}
                    <span className='font-medium text-foreground'>{busiest.registrationCount.toLocaleString()}</span>{' '}
                    for events on {formatDate(busiest.date)}.
                  </p>
                )}
              </>
            ) : (
              <p className='py-12 text-center text-sm text-muted-foreground'>No activity in this period</p>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
