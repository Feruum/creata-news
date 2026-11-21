'use client'

import { PieChartIcon } from 'lucide-react'
import * as React from 'react'
import { Label, Pie, PieChart, Sector } from 'recharts'
import type { PieSectorDataItem } from 'recharts/types/polar/Pie'
import { useEventCategoriesDistributionByMonth } from '@/features/statistics'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import type { ChartConfig } from '@/shared/ui/components/chart'
import { ChartContainer, ChartStyle, ChartTooltip, ChartTooltipContent } from '@/shared/ui/components/chart'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { Skeleton } from '@/shared/ui/components/skeleton'

const MONTHS = [
  { value: 1, label: 'January' },
  { value: 2, label: 'February' },
  { value: 3, label: 'March' },
  { value: 4, label: 'April' },
  { value: 5, label: 'May' },
  { value: 6, label: 'June' },
  { value: 7, label: 'July' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'October' },
  { value: 11, label: 'November' },
  { value: 12, label: 'December' }
]

interface CategoryData {
  categoryId: number
  categoryName: string
  categoryColor: string
  eventCount: number
}

interface ChartDataItem extends CategoryData {
  fill: string
}

function generateChartConfig(categories: CategoryData[]): ChartConfig {
  const config: ChartConfig = {
    eventCount: {
      label: 'Events'
    }
  }

  categories.forEach((category) => {
    config[`category${category.categoryId}`] = {
      label: category.categoryName,
      color: category.categoryColor
    }
  })

  return config
}

interface ChartPieInteractiveProps {
  year?: number
  month?: number
}

export function ChartPieInteractive({ year, month }: ChartPieInteractiveProps) {
  const id = 'pie-interactive'
  const currentDate = new Date()
  const [selectedYear] = React.useState(year ?? currentDate.getFullYear())
  const [selectedMonth, setSelectedMonth] = React.useState(month ?? currentDate.getMonth() + 1)
  const [activeCategoryId, setActiveCategoryId] = React.useState<number | null>(null)

  const { data, isLoading } = useEventCategoriesDistributionByMonth({
    year: selectedYear,
    month: selectedMonth
  })

  const chartData = React.useMemo<ChartDataItem[]>(() => {
    if (!data?.categories) return []
    return data.categories.map((category: CategoryData) => ({
      ...category,
      fill: category.categoryColor
    }))
  }, [data])

  const chartConfig = React.useMemo(
    () => (chartData.length > 0 ? generateChartConfig(data?.categories || []) : {}),
    [data?.categories, chartData.length]
  )

  const activeIndex = React.useMemo(() => {
    if (!activeCategoryId) return 0
    return chartData.findIndex((item) => item.categoryId === activeCategoryId)
  }, [activeCategoryId, chartData])

  React.useEffect(() => {
    if (chartData.length > 0 && !activeCategoryId) {
      setActiveCategoryId(chartData[0].categoryId)
    }
  }, [chartData, activeCategoryId])

  const monthSelector = (
    <Select
      value={selectedMonth.toString()}
      onValueChange={(value) => {
        if (value === null) return
        return setSelectedMonth(Number(value))
      }}
    >
      <SelectTrigger className='ml-auto h-7 w-[130px] rounded-lg pl-2.5' aria-label='Select month'>
        <SelectValue placeholder='Select month' />
      </SelectTrigger>
      <SelectContent align='end' className='rounded-xl'>
        {MONTHS.map((m) => (
          <SelectItem key={m.value} value={m.value.toString()} className='rounded-lg'>
            {m.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )

  if (isLoading) {
    return (
      <Card className='flex flex-col'>
        <CardHeader className='flex-row items-start space-y-0 pb-0'>
          <div className='grid gap-1'>
            <CardTitle className='flex items-center gap-2'>
              <PieChartIcon className='size-5' />
              Event Categories Distribution
            </CardTitle>
            <CardDescription>Loading...</CardDescription>
          </div>
          {monthSelector}
        </CardHeader>
        <CardContent className='flex-1'>
          <Skeleton className='h-[300px] w-full' />
        </CardContent>
      </Card>
    )
  }

  if (!data || chartData.length === 0) {
    return (
      <Card className='flex flex-col'>
        <CardHeader className='flex-row items-start space-y-0 pb-0'>
          <div className='grid gap-1'>
            <CardTitle className='flex items-center gap-2'>
              <PieChartIcon className='size-5' />
              Event Categories Distribution
            </CardTitle>
            <CardDescription>
              {MONTHS.find((m) => m.value === selectedMonth)?.label} {selectedYear}
            </CardDescription>
          </div>
          {monthSelector}
        </CardHeader>
        <CardContent className='flex-1 flex items-center justify-center'>
          <p className='text-muted-foreground'>No events found for this month</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card data-chart={id} className='flex flex-col'>
      <ChartStyle id={id} config={chartConfig} />
      <CardHeader className='flex-row items-start space-y-0 pb-0'>
        <div className='grid gap-1'>
          <CardTitle className='flex items-center gap-2'>
            <PieChartIcon className='size-5' />
            Event Categories Distribution
          </CardTitle>
          <CardDescription>
            {MONTHS.find((m) => m.value === selectedMonth)?.label} {selectedYear} • {data?.totalEvents || 0} events
          </CardDescription>
        </div>
        {monthSelector}
      </CardHeader>
      <CardContent className='flex flex-1 justify-center pb-0'>
        <ChartContainer id={id} config={chartConfig} className='mx-auto aspect-square w-full max-w-[300px]'>
          <PieChart>
            <ChartTooltip cursor={false} content={<ChartTooltipContent hideLabel />} />
            <Pie
              data={chartData}
              dataKey='eventCount'
              nameKey='categoryName'
              innerRadius={60}
              strokeWidth={5}
              activeIndex={activeIndex}
              onClick={(data) => setActiveCategoryId(data.categoryId)}
              activeShape={({ outerRadius = 0, ...props }: PieSectorDataItem) => (
                <g>
                  <Sector {...props} outerRadius={outerRadius + 10} />
                  <Sector {...props} outerRadius={outerRadius + 25} innerRadius={outerRadius + 12} />
                </g>
              )}
            >
              <Label
                content={({ viewBox }) => {
                  if (viewBox && 'cx' in viewBox && 'cy' in viewBox) {
                    const activeCategory = chartData[activeIndex]
                    return (
                      <text x={viewBox.cx} y={viewBox.cy} textAnchor='middle' dominantBaseline='middle'>
                        <tspan x={viewBox.cx} y={viewBox.cy} className='fill-foreground text-3xl font-bold'>
                          {activeCategory?.eventCount || 0}
                        </tspan>
                        <tspan x={viewBox.cx} y={(viewBox.cy || 0) + 24} className='fill-muted-foreground'>
                          Events
                        </tspan>
                      </text>
                    )
                  }
                }}
              />
            </Pie>
          </PieChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
