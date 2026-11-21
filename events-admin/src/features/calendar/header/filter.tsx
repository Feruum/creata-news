import { CheckIcon, Filter, RefreshCcw } from 'lucide-react'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { Separator } from '@/shared/ui/components/separator'
import { Toggle } from '@/shared/ui/components/toggle'

export default function FilterEvents() {
  const { categories, selectedCategoryIds, filterEventsByCategory, clearFilter } = useCalendar()

  if (categories.length === 0) {
    return null
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Toggle variant='outline' className='cursor-pointer w-fit' />}>
        <Filter className='h-4 w-4' />
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-[180px]'>
        {categories.map((category) => (
          <DropdownMenuItem
            key={category.id}
            className='flex items-center gap-2 cursor-pointer'
            onClick={(e) => {
              e.preventDefault()
              filterEventsByCategory(category.id)
            }}
          >
            <div className='size-3.5 rounded-full border' style={{ backgroundColor: category.color }} />
            <span className='flex justify-center items-center gap-2 truncate'>
              {category.name}
              {selectedCategoryIds.includes(category.id) && (
                <span className='text-blue-500'>
                  <CheckIcon className='size-4' />
                </span>
              )}
            </span>
          </DropdownMenuItem>
        ))}
        <Separator className='my-2' />
        <DropdownMenuItem
          disabled={selectedCategoryIds.length === 0}
          className='flex gap-2 cursor-pointer'
          onClick={(e) => {
            e.preventDefault()
            clearFilter()
          }}
        >
          <RefreshCcw className='size-3.5' />
          Clear Filter
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
