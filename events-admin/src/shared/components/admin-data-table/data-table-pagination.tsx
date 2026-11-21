import type { Table as TanstackTable } from '@tanstack/react-table'
import { ChevronFirstIcon, ChevronLastIcon, ChevronLeftIcon, ChevronRightIcon } from 'lucide-react'

import { Button } from '@/shared/ui/components/button'
import { Label } from '@/shared/ui/components/label'
import { Pagination, PaginationContent, PaginationEllipsis, PaginationItem } from '@/shared/ui/components/pagination'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { cn } from '@/shared/ui/lib/utils'

export type DataTablePaginationProps<TData> = {
  table: TanstackTable<TData>
  displayTotalCount: number
  pageSizeOptions: number[]
  id: string
}

export function DataTablePagination<TData>({
  table,
  displayTotalCount,
  pageSizeOptions,
  id
}: DataTablePaginationProps<TData>) {
  const currentPage = table.getState().pagination.pageIndex + 1
  const pageSize = table.getState().pagination.pageSize
  const totalPages = Math.ceil(displayTotalCount / pageSize)

  const pages: (number | 'ellipsis-start' | 'ellipsis-end')[] = []
  const maxVisiblePages = 5

  if (totalPages <= maxVisiblePages + 2) {
    for (let i = 1; i <= totalPages; i++) pages.push(i)
  } else {
    pages.push(1)

    let rangeStart = Math.max(2, currentPage - 1)
    let rangeEnd = Math.min(totalPages - 1, currentPage + 1)

    if (currentPage <= 3) {
      rangeEnd = Math.min(totalPages - 1, 4)
    } else if (currentPage >= totalPages - 2) {
      rangeStart = Math.max(2, totalPages - 3)
    }

    if (rangeStart > 2) pages.push('ellipsis-start')

    for (let i = rangeStart; i <= rangeEnd; i++) pages.push(i)

    if (rangeEnd < totalPages - 1) pages.push('ellipsis-end')

    if (totalPages > 1) pages.push(totalPages)
  }

  return (
    <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
      <div className='flex items-center gap-3'>
        <Label className='text-sm text-muted-foreground whitespace-nowrap' htmlFor={`${id}-pageSize`}>
          Rows per page
        </Label>
        <Select
          onValueChange={(value) => {
            if (value === null) return
            return table.setPageSize(Number(value))
          }}
          value={table.getState().pagination.pageSize.toString()}
        >
          <SelectTrigger className='h-9 w-[70px]' id={`${id}-pageSize`}>
            <SelectValue placeholder='Select' />
          </SelectTrigger>
          <SelectContent side='top'>
            {pageSizeOptions.map((pageSizeOption) => (
              <SelectItem key={pageSizeOption} value={pageSizeOption.toString()}>
                {pageSizeOption}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Pagination className='mx-0 w-auto'>
        <PaginationContent className='gap-1'>
          <PaginationItem>
            <Button
              aria-label='Go to first page'
              className='size-9'
              disabled={!table.getCanPreviousPage()}
              onClick={() => table.firstPage()}
              size='icon'
              variant='outline'
            >
              <ChevronFirstIcon className='size-4' />
            </Button>
          </PaginationItem>

          <PaginationItem>
            <Button
              aria-label='Go to previous page'
              className='size-9'
              disabled={!table.getCanPreviousPage()}
              onClick={() => table.previousPage()}
              size='icon'
              variant='outline'
            >
              <ChevronLeftIcon className='size-4' />
            </Button>
          </PaginationItem>

          {pages.map((page) =>
            typeof page === 'string' ? (
              <PaginationItem key={page}>
                <PaginationEllipsis />
              </PaginationItem>
            ) : (
              <PaginationItem key={page}>
                <Button
                  aria-label={`Go to page ${page}`}
                  aria-current={currentPage === page ? 'page' : undefined}
                  className={cn(
                    'size-9 font-medium',
                    currentPage === page &&
                      'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
                  )}
                  onClick={() => table.setPageIndex(page - 1)}
                  size='icon'
                  variant={currentPage === page ? 'default' : 'outline'}
                >
                  {page}
                </Button>
              </PaginationItem>
            )
          )}

          <PaginationItem>
            <Button
              aria-label='Go to next page'
              className='size-9'
              disabled={!table.getCanNextPage()}
              onClick={() => table.nextPage()}
              size='icon'
              variant='outline'
            >
              <ChevronRightIcon className='size-4' />
            </Button>
          </PaginationItem>

          <PaginationItem>
            <Button
              aria-label='Go to last page'
              className='size-9'
              disabled={!table.getCanNextPage()}
              onClick={() => table.lastPage()}
              size='icon'
              variant='outline'
            >
              <ChevronLastIcon className='size-4' />
            </Button>
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}
