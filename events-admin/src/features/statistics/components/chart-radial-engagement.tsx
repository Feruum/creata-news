import { Filter } from 'lucide-react'
import { Cell, Funnel, FunnelChart, ResponsiveContainer, Tooltip } from 'recharts'
import { useUserEngagementLevels } from '@/features/statistics'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/shared/ui/components/card'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { cn } from '@/shared/ui/lib/utils'
import { formatFunnelLabel, formatLevelName } from '../utils/funnel-helpers'

const FUNNEL_COLORS = ['#10b981', '#8b5cf6', '#ec4899', '#f97316', '#eab308']

export type EngagementLevel = 'active' | 'registered' | 'attended' | 'repeat'

interface FunnelDataItem {
  name: string
  level: string
  value: number
  percentage: number
  fill: string
  microConversion?: number
  dropOff?: number
}

interface ChartRadialEngagementProps {
  selectedStage?: EngagementLevel | null
  onStageSelect?: (stage: EngagementLevel | null) => void
}

interface CustomTooltipProps {
  active?: boolean
  payload?: Array<{
    payload: FunnelDataItem
  }>
}

function CustomTooltip({ active, payload }: CustomTooltipProps) {
  if (!active || !payload || !payload.length) return null

  const data = payload[0].payload

  return (
    <div className='min-w-[180px] rounded-xl border border-border bg-card px-4 py-3'>
      <div className='mb-2.5 border-b border-border pb-2 text-sm font-semibold text-foreground'>{data.name}</div>

      <div className='space-y-2'>
        <div className='flex items-center justify-between gap-4'>
          <span className='text-xs text-muted-foreground'>Users</span>
          <span className='font-mono text-sm font-semibold tabular-nums text-foreground'>
            {data.value.toLocaleString()}
          </span>
        </div>
        <div className='flex items-center justify-between gap-4'>
          <span className='text-xs text-muted-foreground'>Of total</span>
          <span className='font-mono text-sm font-semibold tabular-nums text-foreground'>
            {data.percentage.toFixed(1)}%
          </span>
        </div>
        {data.microConversion !== undefined && (
          <>
            <div className='flex items-center justify-between gap-4'>
              <span className='text-xs text-muted-foreground'>Converted</span>
              <span className='font-mono text-sm font-semibold tabular-nums text-success'>
                +{data.microConversion.toFixed(1)}%
              </span>
            </div>
            <div className='flex items-center justify-between gap-4'>
              <span className='text-xs text-muted-foreground'>Drop-off</span>
              <span className='font-mono text-sm font-semibold tabular-nums text-destructive'>
                -{data.dropOff?.toFixed(1)}%
              </span>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

export function ChartRadialEngagement({ selectedStage, onStageSelect }: ChartRadialEngagementProps) {
  const { data, isLoading } = useUserEngagementLevels()

  const handleStageClick = (level: string) => {
    if (!onStageSelect) return
    const normalizedLevel = level.toLowerCase() as EngagementLevel
    if (selectedStage === normalizedLevel) {
      onStageSelect(null)
    } else {
      onStageSelect(normalizedLevel)
    }
  }

  if (isLoading) {
    return (
      <Card className='flex flex-col'>
        <CardHeader className='pb-2'>
          <CardTitle className='flex items-center gap-2'>
            <Filter className='size-5' />
            Engagement Funnel
          </CardTitle>
          <CardDescription>User progression from signup to repeat attendance</CardDescription>
        </CardHeader>
        <CardContent className='flex-1 pb-0'>
          <Skeleton className='h-[300px] w-full' />
        </CardContent>
      </Card>
    )
  }

  const levels = data?.levels || []

  // Calculate overall conversion directly from funnel data (last / first * 100)
  const firstCount = levels.length > 0 ? levels[0].count : 0
  const lastCount = levels.length > 0 ? levels[levels.length - 1].count : 0
  const overallConversion = firstCount > 0 ? (lastCount / firstCount) * 100 : 0

  if (levels.length === 0) {
    return (
      <Card className='flex flex-col'>
        <CardHeader className='pb-2'>
          <CardTitle className='flex items-center gap-2'>
            <Filter className='size-5' />
            Engagement Funnel
          </CardTitle>
          <CardDescription>No engagement data available</CardDescription>
        </CardHeader>
        <CardContent className='flex-1 pb-0 flex items-center justify-center'>
          <p className='text-sm text-muted-foreground'>No user engagement data to display</p>
        </CardContent>
      </Card>
    )
  }

  const funnelData: FunnelDataItem[] = levels.map(
    (level: { level: string; count: number; percentage: number }, index: number) => {
      const prevCount = index > 0 ? levels[index - 1].count : level.count
      const microConversion = index > 0 && prevCount > 0 ? (level.count / prevCount) * 100 : undefined
      const dropOff = microConversion !== undefined ? 100 - microConversion : undefined

      return {
        name: formatLevelName(level.level),
        level: level.level,
        value: level.count,
        percentage: level.percentage,
        fill: FUNNEL_COLORS[index % FUNNEL_COLORS.length],
        microConversion,
        dropOff
      }
    }
  )

  return (
    <Card className='flex flex-col'>
      <CardHeader className='pb-0'>
        <CardTitle className='flex items-center gap-2'>
          <Filter className='size-5' />
          Engagement Funnel
        </CardTitle>
        <CardDescription>User progression from signup to repeat attendance</CardDescription>
      </CardHeader>

      <CardContent className='flex-1 pt-2 pb-3'>
        <div
          role='img'
          aria-label={`Engagement funnel showing ${funnelData.length} stages: ${funnelData.map((d) => `${d.name} with ${d.value} users`).join(', ')}`}
          className='rounded-lg'
        >
          <ResponsiveContainer width='100%' height={180}>
            <FunnelChart>
              <Tooltip content={<CustomTooltip />} />
              <Funnel dataKey='value' data={funnelData} isAnimationActive stroke='none'>
                {funnelData.map((entry) => (
                  <Cell
                    key={`cell-${entry.level}`}
                    fill={
                      FUNNEL_COLORS[funnelData.findIndex((item) => item.level === entry.level) % FUNNEL_COLORS.length]
                    }
                    opacity={selectedStage && selectedStage !== entry.level.toLowerCase() ? 0.4 : 1}
                    style={{ cursor: onStageSelect ? 'pointer' : 'default' }}
                    onClick={() => handleStageClick(entry.level)}
                  />
                ))}
              </Funnel>
            </FunnelChart>
          </ResponsiveContainer>
        </div>

        <div className='mt-2 space-y-0.5'>
          {funnelData.map((entry, index) => {
            const isSelected = selectedStage === entry.level.toLowerCase()
            const isDimmed = selectedStage && !isSelected

            return (
              <button
                key={entry.name}
                type='button'
                onClick={() => handleStageClick(entry.level)}
                className={cn(
                  'flex items-center gap-2 rounded px-2 py-1 transition-colors text-left w-full',
                  onStageSelect && 'cursor-pointer hover:bg-muted/50',
                  isSelected && 'bg-primary/10 ring-1 ring-primary/20',
                  isDimmed && 'opacity-50'
                )}
              >
                <div
                  className='size-2.5 shrink-0 rounded-sm'
                  style={{ backgroundColor: FUNNEL_COLORS[index % FUNNEL_COLORS.length] }}
                  aria-hidden='true'
                />
                <span className='text-sm font-medium flex-1'>{entry.name}</span>
                <span className='text-sm text-muted-foreground tabular-nums'>
                  {formatFunnelLabel(entry.value, entry.percentage)}
                </span>
                {entry.microConversion !== undefined && (
                  <span className='text-xs tabular-nums text-muted-foreground/70 ml-1'>
                    ← {entry.microConversion.toFixed(0)}%
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </CardContent>

      <CardFooter className='text-sm border-t pt-3 pb-3'>
        <div className='flex items-center gap-2 text-muted-foreground'>
          <span className='tabular-nums font-semibold text-foreground'>{overallConversion.toFixed(1)}%</span>
          <span>overall conversion</span>
          <span className='text-xs'>
            ({funnelData[0]?.name} → {funnelData[funnelData.length - 1]?.name})
          </span>
        </div>
      </CardFooter>
    </Card>
  )
}
