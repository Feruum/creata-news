'use client'

import type { Table } from '@tanstack/react-table'
import { Settings2Icon } from 'lucide-react'

import { Button } from '@/shared/ui/components/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'

interface DataTableViewOptionsProps<TData> {
  /** Table instance from TanStack Table */
  table: Table<TData>
  /** Custom column name formatter */
  formatColumnName?: (columnId: string) => string
}

/**
 * Formats a column ID into a human-readable title
 * Converts camelCase and snake_case to Title Case
 */
function defaultFormatColumnName(columnId: string): string {
  return columnId
    .replace(/([A-Z])/g, ' $1')
    .replace(/[_-]/g, ' ')
    .replace(/^\s+/, '')
    .split(' ')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ')
}

export function DataTableViewOptions<TData>({
  table,
  formatColumnName = defaultFormatColumnName
}: DataTableViewOptionsProps<TData>) {
  const hidableColumns = table.getAllColumns().filter((column) => column.getCanHide())

  if (hidableColumns.length === 0) {
    return null
  }

  const visibleCount = hidableColumns.filter((col) => col.getIsVisible()).length
  const hiddenCount = hidableColumns.length - visibleCount
  const isLastVisible = visibleCount === 1

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant='outline' size='sm' className='h-9' />}>
        <Settings2Icon className='mr-2 size-4' />
        View
        {hiddenCount > 0 && (
          <span className='ml-2 rounded-md bg-muted px-1.5 py-0.5 text-xs font-medium'>{hiddenCount} hidden</span>
        )}
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end' className='w-48'>
        <DropdownMenuGroup>
          <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        {hidableColumns.map((column) => (
          <DropdownMenuCheckboxItem
            key={column.id}
            checked={column.getIsVisible()}
            disabled={isLastVisible && column.getIsVisible()}
            onCheckedChange={(value) => column.toggleVisibility(!!value)}
            onSelect={(e) => e.preventDefault()}
          >
            {formatColumnName(column.id)}
          </DropdownMenuCheckboxItem>
        ))}
        {hiddenCount > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => {
                hidableColumns.forEach((col) => {
                  col.toggleVisibility(true)
                })
              }}
            >
              Show all
            </DropdownMenuItem>
          </>
        )}
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel className='text-xs font-normal text-muted-foreground'>
            {visibleCount} of {hidableColumns.length} visible
          </DropdownMenuLabel>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
