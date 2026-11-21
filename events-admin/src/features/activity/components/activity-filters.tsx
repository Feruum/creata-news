import { ListFilter, X } from 'lucide-react'
import type { DateRange } from 'react-day-picker'
import { Button } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { DateRangePicker } from '@/shared/ui/components/date-range-picker'
import { Label } from '@/shared/ui/components/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { ACTION_GROUPS, getActionLabel } from './action-display'

export function ActionFilterPopover({ value, onChange }: { value: string[]; onChange: (v: string[]) => void }) {
  const toggle = (action: string) => {
    if (value.includes(action)) {
      onChange(value.filter((a) => a !== action))
    } else {
      onChange([...value, action])
    }
  }

  return (
    <Popover>
      <PopoverTrigger render={<Button variant='outline' size='sm' className='gap-1.5' />}>
        <ListFilter className='size-4' />
        Actions
        {value.length > 0 && (
          <span className='ml-1 rounded-full bg-primary/10 px-1.5 text-xs font-medium'>{value.length}</span>
        )}
      </PopoverTrigger>
      <PopoverContent className='w-72 p-3' align='start'>
        <div className='flex items-center justify-between pb-2'>
          <span className='text-sm font-medium'>Filter by action</span>
          {value.length > 0 && (
            <Button variant='ghost' size='sm' onClick={() => onChange([])}>
              <X className='size-3.5' />
              Clear
            </Button>
          )}
        </div>
        <div className='max-h-64 space-y-3 overflow-y-auto'>
          {Object.entries(ACTION_GROUPS).map(([group, actions]) => (
            <div key={group}>
              <span className='text-xs font-medium text-muted-foreground uppercase tracking-wider'>{group}</span>
              <div className='mt-1 space-y-1'>
                {actions.map((action) => (
                  <div key={action} className='flex items-center gap-2'>
                    <Checkbox id={action} checked={value.includes(action)} onCheckedChange={() => toggle(action)} />
                    <Label htmlFor={action} className='text-sm font-normal cursor-pointer'>
                      {getActionLabel(action)}
                    </Label>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}

export function DateRangeFilter({
  after,
  before,
  onAfterChange,
  onBeforeChange
}: {
  after: string | null
  before: string | null
  onAfterChange: (v: string | null) => void
  onBeforeChange: (v: string | null) => void
}) {
  const value: DateRange | undefined =
    after || before
      ? {
          from: after ? new Date(after) : undefined,
          to: before ? new Date(before) : undefined
        }
      : undefined

  return (
    <DateRangePicker
      value={value}
      onChange={(range) => {
        onAfterChange(range?.from?.toISOString() ?? null)
        onBeforeChange(range?.to?.toISOString() ?? null)
      }}
    />
  )
}
