import type { Category } from '@studyverse/proto'
import type { ColumnDef, FilterFn } from '@tanstack/react-table'

import { formatDate } from '@/shared/lib/utils'
import { CategoryActionsCell } from './components/category-actions-cell'

export const multiColumnFilterFn: FilterFn<Category> = (row, _columnId, filterValue) => {
  const searchableRowContent = `${row.original.name}`.toLowerCase()
  const searchTerm = (filterValue ?? '').toLowerCase()
  return searchableRowContent.includes(searchTerm)
}

interface CreateColumnsOptions {
  onEdit: (category: Category) => void
}

export function createColumns({ onEdit }: CreateColumnsOptions): ColumnDef<Category>[] {
  return [
    {
      accessorKey: 'id',
      header: 'ID',
      size: 70,
      minSize: 60,
      maxSize: 80,
      enableSorting: true,
      cell: ({ row }) => <span className='font-mono text-xs text-muted-foreground'>{row.getValue('id')}</span>
    },
    {
      accessorKey: 'name',
      header: 'Category Name',
      size: 200,
      minSize: 120,
      enableSorting: true,
      filterFn: multiColumnFilterFn,
      cell: ({ row }) => (
        <div className='flex items-center gap-2'>
          <div className='size-3 shrink-0 rounded-full border' style={{ backgroundColor: row.original.color }} />
          <span className='font-medium text-sm truncate'>{row.getValue('name')}</span>
        </div>
      )
    },
    {
      accessorKey: 'color',
      header: 'Color',
      size: 120,
      minSize: 100,
      enableSorting: true,
      cell: ({ row }) => {
        const color = row.getValue('color') as string
        return (
          <div className='flex items-center gap-2'>
            <div className='size-4 rounded-full border' style={{ backgroundColor: color }} />
            <span className='font-mono text-xs text-muted-foreground uppercase'>{color}</span>
          </div>
        )
      }
    },
    {
      accessorKey: 'createdAt',
      header: 'Created',
      size: 120,
      minSize: 100,
      enableSorting: true,
      cell: ({ row }) => (
        <span className='text-xs text-muted-foreground tabular-nums whitespace-nowrap'>
          {formatDate(row.getValue('createdAt') as string)}
        </span>
      )
    },
    {
      accessorKey: 'updatedAt',
      header: 'Updated',
      size: 120,
      minSize: 100,
      enableSorting: true,
      cell: ({ row }) => (
        <span className='text-xs text-muted-foreground tabular-nums whitespace-nowrap'>
          {formatDate(row.getValue('updatedAt') as string)}
        </span>
      )
    },
    {
      id: 'actions',
      size: 50,
      cell: ({ row }) => <CategoryActionsCell category={row.original} onEdit={onEdit} />
    }
  ]
}
