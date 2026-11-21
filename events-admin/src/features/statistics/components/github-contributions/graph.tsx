'use client'

import { format, parseISO } from 'date-fns'
import { LoaderIcon } from 'lucide-react'
import { useMemo } from 'react'
import {
  type Activity,
  ContributionGraph,
  ContributionGraphCalendar,
  ContributionGraphFooter,
  ContributionGraphLegend,
  ContributionGraphTotalCount
} from '@/features/statistics/components/kibo-ui/contribution-graph'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/components/tooltip'
import { fillActivityYear } from './year-activity'

const colors = [
  'fill-[#eeebf7] dark:fill-[#302d40]',
  'fill-[#edf6ba] dark:fill-[#525e32]',
  'fill-[#d9ee87] dark:fill-[#829d46]',
  'fill-[#c9e960] dark:fill-[#c9e960]',
  'fill-[#91ac3e] dark:fill-[#a6bf58]'
]

const blockSize = 14
const blockMargin = 4
const blockRadius = 4
const fontSize = 10

export function AnimatedGitHubContributionGraph({ contributions, year }: { contributions: Activity[]; year: number }) {
  const data = useMemo(() => fillActivityYear(contributions, year), [contributions, year])

  return (
    <ContributionGraph
      className='w-full min-w-0 gap-4'
      data={data}
      blockSize={blockSize}
      blockMargin={blockMargin}
      blockRadius={blockRadius}
      fontSize={fontSize}
    >
      <ContributionGraphCalendar
        key={year}
        monthLabelPosition='bottom'
        className='rounded-md pb-2 text-muted-foreground uppercase outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset [&>svg]:min-w-full'
        tabIndex={0}
        role='region'
        aria-label={`Event activity for ${year}, scroll to view all months`}
      >
        {({ activity, dayIndex, weekIndex }) => (
          <Tooltip key={activity.date}>
            <TooltipTrigger render={<g />}>
              <rect
                className={`${colors[activity.level] ?? colors[0]} transition-opacity hover:opacity-70`}
                data-date={activity.date}
                data-count={activity.count}
                width={blockSize}
                height={blockSize}
                rx={blockRadius}
                ry={blockRadius}
                x={(blockSize + blockMargin) * weekIndex}
                y={fontSize + 8 + (blockSize + blockMargin) * dayIndex}
              />
            </TooltipTrigger>
            <TooltipContent>
              {activity.count} event{activity.count !== 1 ? 's' : ''} on {format(parseISO(activity.date), 'dd.MM.yyyy')}
            </TooltipContent>
          </Tooltip>
        )}
      </ContributionGraphCalendar>
      <ContributionGraphFooter className='items-center justify-between gap-x-4 gap-y-2 text-xs'>
        <ContributionGraphTotalCount>
          {({ totalCount }) => (
            <span className='text-muted-foreground'>
              <strong className='font-medium text-foreground'>{totalCount}</strong> events in {year}
            </span>
          )}
        </ContributionGraphTotalCount>
        <ContributionGraphLegend>
          {({ level }) => (
            <svg width={12} height={12} aria-hidden='true'>
              <rect className={colors[level]} width={12} height={12} rx={3} />
            </svg>
          )}
        </ContributionGraphLegend>
      </ContributionGraphFooter>
    </ContributionGraph>
  )
}

export function GitHubContributionFallback() {
  return (
    <div
      role='status'
      aria-label='Loading event activity'
      className='flex h-[190px] w-full items-center justify-center'
    >
      <LoaderIcon className='animate-spin text-muted-foreground' />
    </div>
  )
}
