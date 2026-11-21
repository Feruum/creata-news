import { CheckIcon } from 'lucide-react'

import { cn } from '@/shared/ui/lib/utils'

const CATEGORY_COLORS = [
  { name: 'Academic', color: '#0693E3' },
  { name: 'Social', color: '#FF6900' },
  { name: 'Creative', color: '#9900EF' },
  { name: 'Administrative', color: '#ABB8C3' },
  { name: 'Sport', color: '#00D084' },
  { name: 'Science', color: '#0288D1' },
  { name: 'Career', color: '#FCB900' },
  { name: 'Pink', color: '#F78DA7' },
  { name: 'Red', color: '#EB144C' }
] as const

interface CategoryColorPickerProps {
  value?: string
  onChange: (color: string) => void
  className?: string
}

export function CategoryColorPicker({ value, onChange, className }: CategoryColorPickerProps) {
  return (
    <div className={cn('flex flex-wrap gap-2', className)}>
      {CATEGORY_COLORS.map((item) => {
        const isSelected = value === item.color
        return (
          <button
            key={item.color}
            type='button'
            onClick={() => onChange(item.color)}
            className={cn(
              'group relative flex size-8 items-center justify-center rounded-full border-2 transition-all hover:scale-110 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
              isSelected ? 'border-foreground' : 'border-transparent'
            )}
            style={{ backgroundColor: item.color }}
            title={item.name}
          >
            {isSelected && <CheckIcon className='size-4 text-white' />}
          </button>
        )
      })}
    </div>
  )
}

export { CATEGORY_COLORS }
