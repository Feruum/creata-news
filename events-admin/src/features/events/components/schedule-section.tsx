import { format } from 'date-fns'
import { CalendarIcon } from 'lucide-react'
import type { UseFormReturn } from 'react-hook-form'
import { Button } from '@/shared/ui/components/button'
import { Calendar } from '@/shared/ui/components/calendar'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/shared/ui/components/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { Switch } from '@/shared/ui/components/switch'
import { TimePicker } from '@/shared/ui/components/time-picker'
import { cn } from '@/shared/ui/lib/utils'
import type { CreateEventFormData, CreateEventFormInput } from '../schema'
import { isEventCalendarDateDisabled } from '../utils/event-date-helpers'
import { SectionHeader } from './section-header'

interface ScheduleSectionProps {
  form: UseFormReturn<CreateEventFormInput, unknown, CreateEventFormData>
}

export function ScheduleSection({ form }: ScheduleSectionProps) {
  const watchHasTime = form.watch('hasTime')

  return (
    <fieldset className='space-y-4 rounded-lg border bg-muted/30 p-4'>
      <legend className='sr-only'>Schedule</legend>
      <div className='flex items-center justify-between'>
        <SectionHeader title='Schedule' description='Set date and optional time' />
        <FormField
          control={form.control}
          name='hasTime'
          render={({ field }) => (
            <FormItem className='flex items-center gap-2'>
              <FormLabel className='text-xs text-muted-foreground'>Include time</FormLabel>
              <FormControl>
                <Switch checked={field.value} onCheckedChange={field.onChange} />
              </FormControl>
            </FormItem>
          )}
        />
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <FormField
          control={form.control}
          name='startDate'
          render={({ field }) => (
            <FormItem className='flex flex-col'>
              <FormLabel>Start Date</FormLabel>
              <Popover>
                <PopoverTrigger render={<FormControl />}>
                  <Button
                    variant='outline'
                    className={cn(
                      'w-full justify-start text-left font-normal',
                      !field.value && 'text-muted-foreground'
                    )}
                  >
                    {field.value ? format(field.value, 'PP') : 'Pick date…'}
                    <CalendarIcon className='ml-auto h-4 w-4 opacity-50' />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className='w-auto p-0' align='start'>
                  <Calendar
                    mode='single'
                    selected={field.value}
                    onSelect={field.onChange}
                    disabled={(date) => isEventCalendarDateDisabled(date)}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={form.control}
          name='endDate'
          render={({ field }) => (
            <FormItem className='flex flex-col'>
              <FormLabel>End Date</FormLabel>
              <Popover>
                <PopoverTrigger render={<FormControl />}>
                  <Button
                    variant='outline'
                    className={cn(
                      'w-full justify-start text-left font-normal',
                      !field.value && 'text-muted-foreground'
                    )}
                  >
                    {field.value ? format(field.value, 'PP') : 'Pick date…'}
                    <CalendarIcon className='ml-auto h-4 w-4 opacity-50' />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className='w-auto p-0' align='start'>
                  <Calendar
                    mode='single'
                    selected={field.value}
                    onSelect={field.onChange}
                    disabled={(date) => {
                      const startDate = form.getValues('startDate')
                      return isEventCalendarDateDisabled(date, startDate)
                    }}
                    initialFocus
                  />
                </PopoverContent>
              </Popover>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>

      {watchHasTime && (
        <div className='grid gap-4 sm:grid-cols-2'>
          <FormField
            control={form.control}
            name='startTime'
            render={({ field }) => (
              <FormItem>
                <FormLabel>Start Time</FormLabel>
                <FormControl>
                  <TimePicker value={field.value} onChange={field.onChange} placeholder='Select start time' />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='endTime'
            render={({ field }) => (
              <FormItem>
                <FormLabel>End Time</FormLabel>
                <FormControl>
                  <TimePicker value={field.value} onChange={field.onChange} placeholder='Select end time' />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}
    </fieldset>
  )
}
