'use client'

import { format, startOfYear, subDays } from 'date-fns'
import { CalendarIcon } from 'lucide-react'
import * as React from 'react'
import type { DateRange } from 'react-day-picker'
import { Button } from '@/shared/ui/components/button'
import { Calendar } from '@/shared/ui/components/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { cn } from '@/shared/ui/lib/utils'

const presets = [
  { label: 'Last 7 days', getValue: () => ({ from: subDays(new Date(), 7), to: new Date() }) },
  { label: 'Last 30 days', getValue: () => ({ from: subDays(new Date(), 30), to: new Date() }) },
  { label: 'Last 90 days', getValue: () => ({ from: subDays(new Date(), 90), to: new Date() }) },
  { label: 'This year', getValue: () => ({ from: startOfYear(new Date()), to: new Date() }) }
]

interface DateRangePickerProps {
  value?: DateRange
  onChange?: (range: DateRange | undefined) => void
  className?: string
  align?: 'start' | 'center' | 'end'
}

function DateRangePicker({ value, onChange, className, align = 'start' }: DateRangePickerProps) {
  const [date, setDate] = React.useState<DateRange | undefined>(value)

  const handleSelect = (range: DateRange | undefined) => {
    setDate(range)
    onChange?.(range)
  }

  const handlePreset = (preset: (typeof presets)[0]) => {
    const range = preset.getValue()
    setDate(range)
    onChange?.(range)
  }

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            className={cn(
              'justify-start text-left font-normal h-8 px-3 text-xs',
              !date && 'text-muted-foreground',
              className
            )}
          />
        }
      >
        <CalendarIcon className='mr-2 h-3.5 w-3.5' />
        {date?.from ? (
          date.to ? (
            <>
              {format(date.from, 'MMM d')} - {format(date.to, 'MMM d, yyyy')}
            </>
          ) : (
            format(date.from, 'MMM d, yyyy')
          )
        ) : (
          <span>Select date range</span>
        )}
      </PopoverTrigger>
      <PopoverContent className='w-auto p-0' align={align}>
        <div className='flex'>
          <div className='flex flex-col gap-1 border-r p-2'>
            {presets.map((preset) => (
              <Button
                key={preset.label}
                variant='ghost'
                size='sm'
                className='justify-start text-xs h-7'
                onClick={() => handlePreset(preset)}
              >
                {preset.label}
              </Button>
            ))}
          </div>
          <Calendar mode='range' defaultMonth={date?.from} selected={date} onSelect={handleSelect} numberOfMonths={2} />
        </div>
      </PopoverContent>
    </Popover>
  )
}

export type { DateRangePickerProps }
export { DateRangePicker }
