'use client'

import { enUS } from 'date-fns/locale'
import type { ComponentProps } from 'react'
import { getDefaultClassNames, DayPicker as ReactDayPicker } from 'react-day-picker'
import { buttonVariants } from '@/shared/ui/components/button'
import { cn } from '@/shared/ui/lib/utils'

type TDayPickerProps = ComponentProps<typeof ReactDayPicker>

function DayPicker({ className, classNames, showOutsideDays = true, ...props }: TDayPickerProps) {
  const defaultClassNames = getDefaultClassNames()

  return (
    <ReactDayPicker
      showOutsideDays={showOutsideDays}
      className={cn('p-3 bg-background', className)}
      classNames={{
        root: cn('w-fit', defaultClassNames.root),
        months: cn(
          'flex flex-col select-none sm:flex-row space-y-4 sm:space-x-4 sm:space-y-0',
          defaultClassNames.months
        ),
        month: cn('space-y-4', defaultClassNames.month),
        month_caption: cn(
          'flex items-center justify-center h-8 w-full relative capitalize',
          defaultClassNames.month_caption
        ),
        caption_label: cn('text-sm font-medium', defaultClassNames.caption_label),
        nav: cn('hidden', defaultClassNames.nav),
        weekdays: cn('flex', defaultClassNames.weekdays),
        weekday: cn('w-9 font-medium text-sm text-muted-foreground capitalize', defaultClassNames.weekday),
        week: cn('flex w-full mt-2', defaultClassNames.week),
        week_number: cn(
          'w-9 text-xs text-muted-foreground/60 font-medium flex items-center justify-center',
          defaultClassNames.week_number
        ),
        week_number_header: cn(
          'w-9 text-xs text-muted-foreground/60 font-medium',
          defaultClassNames.week_number_header
        ),
        day: cn(
          'size-9 flex items-center justify-center text-muted-foreground text-center text-sm p-0 relative',
          'focus-within:relative focus-within:z-20',
          '[&:has([aria-selected].range_end)]:rounded-r-md',
          '[&:last-child:has([aria-selected])]:rounded-r-md',
          '[&:first-child:has([aria-selected])]:rounded-l-md',
          '[&:has([aria-selected])]:bg-accent [&:has([aria-selected])]:rounded-md',
          defaultClassNames.day
        ),
        day_button: cn(
          buttonVariants({ variant: 'ghost' }),
          'size-8 p-0 font-normal rounded-md aria-selected:opacity-100',
          defaultClassNames.day_button
        ),
        selected: cn(
          'rounded-md bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground',
          defaultClassNames.selected
        ),
        today: cn('text-primary font-semibold', defaultClassNames.today),
        outside: cn('opacity-50 aria-selected:opacity-40', defaultClassNames.outside),
        range_middle: cn(
          'aria-selected:bg-accent aria-selected:text-accent-foreground',
          defaultClassNames.range_middle
        ),
        hidden: cn('invisible', defaultClassNames.hidden),
        disabled: cn('text-muted-foreground opacity-50', defaultClassNames.disabled),
        ...classNames
      }}
      locale={enUS}
      {...props}
    />
  )
}

export { DayPicker }
