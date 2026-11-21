'use client'

import { ClockIcon } from 'lucide-react'
import * as React from 'react'

import { Button } from '@/shared/ui/components/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { ScrollArea } from '@/shared/ui/components/scroll-area'
import { cn } from '@/shared/ui/lib/utils'

interface TimePickerProps {
  value?: string // "HH:mm" format
  onChange?: (value: string) => void
  placeholder?: string
  disabled?: boolean
  className?: string
  use24Hour?: boolean
}

export function TimePicker({
  value,
  onChange,
  placeholder = 'Select time',
  disabled = false,
  className,
  use24Hour = true
}: TimePickerProps) {
  const [open, setOpen] = React.useState(false)

  // Parse current value
  const [selectedHour, selectedMinute] = React.useMemo(() => {
    if (!value) return [null, null]
    const [h, m] = value.split(':').map(Number)
    return [h, m]
  }, [value])

  const handleTimeSelect = (hour: number, minute: number) => {
    const timeStr = `${hour.toString().padStart(2, '0')}:${minute.toString().padStart(2, '0')}`
    onChange?.(timeStr)
    setOpen(false)
  }

  const formatDisplayTime = (timeValue: string | undefined) => {
    if (!timeValue) return null
    const [h, m] = timeValue.split(':').map(Number)
    if (use24Hour) {
      return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`
    }
    const period = h >= 12 ? 'PM' : 'AM'
    const hour12 = h % 12 || 12
    return `${hour12}:${m.toString().padStart(2, '0')} ${period}`
  }

  const hours = use24Hour ? Array.from({ length: 24 }, (_, i) => i) : Array.from({ length: 12 }, (_, i) => i + 1)

  const minutes = Array.from({ length: 12 }, (_, i) => i * 5)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant='outline'
            disabled={disabled}
            className={cn('w-full justify-start text-left font-normal', !value && 'text-muted-foreground', className)}
          />
        }
      >
        <ClockIcon className='mr-2 h-4 w-4' />
        {formatDisplayTime(value) || placeholder}
      </PopoverTrigger>
      <PopoverContent className='w-auto p-0' align='start'>
        <div className='flex'>
          {/* Hours */}
          <div className='border-r'>
            <div className='px-3 py-2 text-xs font-medium text-muted-foreground border-b'>Hour</div>
            <ScrollArea className='h-[200px]'>
              <div className='flex flex-col p-1'>
                {hours.map((hour) => {
                  const displayHour = use24Hour ? hour : hour
                  const actualHour = use24Hour ? hour : hour === 12 ? 12 : hour
                  return (
                    <Button
                      key={hour}
                      variant={selectedHour === actualHour ? 'default' : 'ghost'}
                      size='sm'
                      className='w-full justify-center'
                      onClick={() => handleTimeSelect(actualHour, selectedMinute ?? 0)}
                    >
                      {displayHour.toString().padStart(2, '0')}
                    </Button>
                  )
                })}
              </div>
            </ScrollArea>
          </div>
          {/* Minutes */}
          <div>
            <div className='px-3 py-2 text-xs font-medium text-muted-foreground border-b'>Min</div>
            <ScrollArea className='h-[200px]'>
              <div className='flex flex-col p-1'>
                {minutes.map((minute) => (
                  <Button
                    key={minute}
                    variant={selectedMinute === minute ? 'default' : 'ghost'}
                    size='sm'
                    className='w-full justify-center'
                    onClick={() => handleTimeSelect(selectedHour ?? 9, minute)}
                  >
                    {minute.toString().padStart(2, '0')}
                  </Button>
                ))}
              </div>
            </ScrollArea>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
