import { useState } from 'react'
import { useEventActivityByYear } from '@/features/statistics'
import type { Activity } from '@/features/statistics/components/kibo-ui/contribution-graph'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { ToggleGroup, ToggleGroupItem } from '@/shared/ui/components/toggle-group'
import { AnimatedGitHubContributionGraph, GitHubContributionFallback } from './graph'

interface EventActivityHeatmapProps {
  year?: number
}

export function EventActivityHeatmap({ year: initialYear }: EventActivityHeatmapProps) {
  const currentYear = new Date().getFullYear()
  const [selectedYear, setSelectedYear] = useState(initialYear ?? currentYear)

  // Generate list of years (current year and 4 previous years)
  const years = Array.from({ length: 5 }, (_, i) => currentYear - i)

  const { data, isLoading } = useEventActivityByYear({ year: selectedYear })

  const activities = (data?.activities || []) as Activity[]
  const contributions = activities.map((activity) => ({
    date: activity.date,
    count: activity.count,
    level: activity.level
  }))

  return (
    <Card className='min-w-0 gap-5 rounded-3xl border-0 bg-white py-5 shadow-sm dark:bg-card sm:py-6'>
      <CardHeader className='flex flex-row flex-wrap items-center justify-between gap-4 px-5 sm:px-6'>
        <div className='space-y-1.5'>
          <CardTitle className='text-base font-medium'>Activity Heatmap</CardTitle>
          <CardDescription className='text-xs'>Events per day over the year</CardDescription>
        </div>

        <ToggleGroup
          multiple={false}
          value={[String(selectedYear)]}
          onValueChange={(value) => value[0] && setSelectedYear(Number(value[0]))}
          aria-label='Activity year'
          className='max-w-full flex-wrap gap-1 rounded-full bg-muted/50 p-1'
          size='sm'
        >
          {years.map((year) => (
            <ToggleGroupItem
              key={year}
              value={String(year)}
              aria-label={`Show activity for ${year}`}
              className='min-w-10 rounded-full px-2.5 text-xs text-muted-foreground data-pressed:bg-[#d9f279] data-pressed:text-[#253019]'
            >
              {year}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </CardHeader>
      <CardContent className='min-w-0 px-5 sm:px-6'>
        {isLoading ? (
          <GitHubContributionFallback />
        ) : (
          <AnimatedGitHubContributionGraph contributions={contributions} year={selectedYear} />
        )}
      </CardContent>
    </Card>
  )
}
