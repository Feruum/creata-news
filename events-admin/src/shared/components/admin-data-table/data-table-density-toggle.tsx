'use client'

import { RowsIcon } from 'lucide-react'

import { Button } from '@/shared/ui/components/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'

export type DensityValue = 'compact' | 'comfortable' | 'spacious'

interface DataTableDensityToggleProps {
  /** Current density value */
  value: DensityValue
  /** Callback when density changes */
  onChange: (value: DensityValue) => void
}

const densityOptions: { value: DensityValue; label: string; description: string }[] = [
  { value: 'compact', label: 'Compact', description: '40px - See more rows' },
  { value: 'comfortable', label: 'Comfortable', description: '52px - Balanced' },
  { value: 'spacious', label: 'Spacious', description: '72px - Easy reading' }
]

export function DataTableDensityToggle({ value, onChange }: DataTableDensityToggleProps) {
  const currentOption = densityOptions.find((opt) => opt.value === value)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant='outline' size='sm' className='h-9' />}>
        <RowsIcon className='mr-2 size-4' />
        {currentOption?.label ?? 'Density'}
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-44'>
        <DropdownMenuGroup>
          <DropdownMenuLabel>Row density</DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuRadioGroup value={value} onValueChange={(v) => onChange(v as DensityValue)}>
          {densityOptions.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
