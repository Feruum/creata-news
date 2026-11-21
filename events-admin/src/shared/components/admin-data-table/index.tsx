'use client'

/* eslint-disable react-refresh/only-export-components */

import {
  type ColumnDef,
  type ColumnFiltersState,
  type ColumnResizeMode,
  flexRender,
  getCoreRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  type Row,
  type RowSelectionState,
  type SortingState,
  type Table as TanstackTable,
  useReactTable,
  type VisibilityState
} from '@tanstack/react-table'
import { useVirtualizer } from '@tanstack/react-virtual'
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CheckIcon,
  ChevronsUpDownIcon,
  CircleXIcon,
  Loader2Icon,
  SearchXIcon,
  Trash2Icon
} from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useId, useMemo, useRef, useState } from 'react'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/shared/ui/components/alert-dialog'
import { Button, buttonVariants } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { cn } from '@/shared/ui/lib/utils'

import { DataTablePagination } from './data-table-pagination'
import { DataTableToolbar, type DensityValue } from './data-table-toolbar'
import { TableRow } from './table-row'
import { useDataTableState } from './use-data-table-state'
import { useKeyboardNavigation } from './use-keyboard-navigation'

export type { DensityValue } from './data-table-toolbar'
export { DataTableDensityToggle, DataTableSearch, DataTableToolbar, DataTableViewOptions } from './data-table-toolbar'
export { FacetedFilter, type FacetedFilterOption, type FacetedFilterProps } from './faceted-filter'
export type { UseDataTableStateOptions } from './use-data-table-state'
export { useDataTableState } from './use-data-table-state'
export { createMultiColumnFilterFn } from './utils'

// ============================================================================
// Types
// ============================================================================

export interface DataTableProps<TData> {
  data: TData[]
  columns: ColumnDef<TData>[]
  totalCount: number
  hasActiveFilters?: boolean
  onClearFilters?: () => void
  isLoading?: boolean
  searchColumnId?: string
  searchPlaceholder?: string
  enableVirtualization?: boolean
  enableDensityToggle?: boolean
  enableMultiSort?: boolean
  enableColumnResizing?: boolean
  defaultSortBy?: string
  defaultSortDesc?: boolean
  entityName?: string
  emptyState?: ReactNode
  toolbarActions?: ReactNode
  filterComponents?: ReactNode
  rowSelection?: RowSelectionState
  onRowSelectionChange?: (selection: RowSelectionState) => void
  onRowClick?: (row: TData) => void
  /** The record open in a preview, independent of bulk row selection. */
  activeRowId?: string | null
  onDeleteSelected?: (ids: (string | number)[]) => void
  /** Whether delete is currently in progress (shows loading state) */
  isDeleting?: boolean
  pageSizeOptions?: number[]
  getRowId?: (row: TData) => string
  /** When true, hides select/delete columns and selection controls (for guest/read-only mode) */
  readOnly?: boolean
}

// ============================================================================
// Component: DataTable
// ============================================================================

export function DataTable<TData extends { id?: string | number }>({
  data,
  columns: baseColumns,
  totalCount,
  hasActiveFilters = false,
  onClearFilters,
  isLoading = false,
  searchColumnId = 'title',
  searchPlaceholder = 'Search...',
  enableVirtualization = false,
  enableDensityToggle = false,
  enableMultiSort = false,
  enableColumnResizing = false,
  defaultSortBy = 'id',
  defaultSortDesc = false,
  entityName = 'items',
  emptyState,
  toolbarActions,
  filterComponents,
  rowSelection: controlledRowSelection,
  onRowSelectionChange,
  onRowClick,
  activeRowId,
  onDeleteSelected,
  isDeleting = false,
  pageSizeOptions = [10, 25, 50, 100],
  getRowId: customGetRowId,
  readOnly = false
}: DataTableProps<TData>) {
  const id = useId()
  const tableContainerRef = useRef<HTMLDivElement>(null)
  const [internalRowSelection, setInternalRowSelection] = useState<RowSelectionState>({})
  const [announcement, setAnnouncement] = useState<string>('')
  const [focusedCell, setFocusedCell] = useState<{ rowIndex: number; colIndex: number } | null>(null)

  // Delete confirmation state
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false)
  const [pendingDeleteIds, setPendingDeleteIds] = useState<(string | number)[]>([])

  // Support controlled or uncontrolled row selection
  const isControlled = controlledRowSelection !== undefined
  const rowSelection = isControlled ? controlledRowSelection : internalRowSelection

  const { tableState, setTableState } = useDataTableState({
    defaultSortBy,
    defaultSortDesc
  })

  const columnFilters: ColumnFiltersState = useMemo(() => {
    const filters: ColumnFiltersState = []
    if (tableState.search) {
      filters.push({ id: searchColumnId, value: tableState.search })
    }
    return filters
  }, [tableState.search, searchColumnId])

  const columnVisibility: VisibilityState = useMemo(() => {
    const visibility: VisibilityState = {}
    tableState.hiddenColumns.forEach((col) => {
      visibility[col] = false
    })
    return visibility
  }, [tableState.hiddenColumns])

  const sorting: SortingState = useMemo(
    () => [{ id: tableState.sortBy, desc: tableState.sortDesc === 1 }],
    [tableState.sortBy, tableState.sortDesc]
  )

  const setColumnFilters = useCallback(
    (updater: ColumnFiltersState | ((old: ColumnFiltersState) => ColumnFiltersState)) => {
      const newFilters = typeof updater === 'function' ? updater(columnFilters) : updater
      const searchFilter = newFilters.find((f) => f.id === searchColumnId)
      setTableState({ search: (searchFilter?.value as string) || null, page: 1 })
    },
    [columnFilters, searchColumnId, setTableState]
  )

  const setColumnVisibility = useCallback(
    (updater: VisibilityState | ((old: VisibilityState) => VisibilityState)) => {
      const newVisibility = typeof updater === 'function' ? updater(columnVisibility) : updater
      const hidden = Object.entries(newVisibility)
        .filter(([, visible]) => !visible)
        .map(([col]) => col)
      setTableState({ hiddenColumns: hidden.length > 0 ? hidden : null })
    },
    [columnVisibility, setTableState]
  )

  const setSorting = useCallback(
    (updater: SortingState | ((old: SortingState) => SortingState)) => {
      const newSorting = typeof updater === 'function' ? updater(sorting) : updater
      if (newSorting.length > 0) {
        setTableState({ sortBy: newSorting[0].id, sortDesc: newSorting[0].desc ? 1 : 0 })
        const direction = newSorting[0].desc ? 'descending' : 'ascending'
        setAnnouncement(`Sorted by ${newSorting[0].id}, ${direction}`)
      }
    },
    [sorting, setTableState]
  )

  const handleRowSelectionChange = useCallback(
    (updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState)) => {
      const newSelection = typeof updater === 'function' ? updater(rowSelection) : updater
      if (!isControlled) {
        setInternalRowSelection(newSelection)
      }
      onRowSelectionChange?.(newSelection)
    },
    [rowSelection, isControlled, onRowSelectionChange]
  )

  const handleDensityChange = useCallback(
    (value: DensityValue) => {
      setTableState({ density: value })
    },
    [setTableState]
  )

  const columns: ColumnDef<TData>[] = useMemo(() => {
    if (readOnly) {
      return baseColumns.filter((col) => col.id !== 'actions')
    }

    return [
      {
        id: 'select',
        size: 48,
        minSize: 48,
        maxSize: 48,
        enableHiding: false,
        enableSorting: false,
        header: () => null,
        cell: ({ row }) => (
          <div className='flex items-center justify-center'>
            <Checkbox
              checked={row.getIsSelected()}
              onCheckedChange={(value) => row.toggleSelected(!!value)}
              onClick={(event) => event.stopPropagation()}
              aria-label='Select row'
              className='data-[state=checked]:bg-primary data-[state=checked]:border-primary'
            />
          </div>
        )
      },
      ...baseColumns.filter((col) => col.id !== 'actions' || !onDeleteSelected),
      ...(onDeleteSelected
        ? [
            {
              id: 'delete',
              size: 48,
              minSize: 48,
              maxSize: 48,
              enableHiding: false,
              enableSorting: false,
              header: () => null,
              cell: ({ row }) => (
                <div className='flex items-center justify-center'>
                  <Button
                    variant='ghost'
                    size='icon'
                    className='h-8 w-8 text-muted-foreground hover:text-destructive hover:bg-destructive/10'
                    onClick={(event) => {
                      event.stopPropagation()
                      const rowId = row.original.id
                      if (rowId !== undefined) {
                        setPendingDeleteIds([rowId])
                        setDeleteConfirmOpen(true)
                      }
                    }}
                    aria-label='Delete row'
                  >
                    <Trash2Icon className='h-4 w-4' />
                  </Button>
                </div>
              )
            } satisfies ColumnDef<TData>
          ]
        : [])
    ]
  }, [baseColumns, onDeleteSelected, readOnly])

  const getRowId = useCallback(
    (row: TData, index: number) => {
      if (customGetRowId) return customGetRowId(row)
      if (row.id !== undefined) return String(row.id)
      return String(index)
    },
    [customGetRowId]
  )

  const isServerSidePagination = !hasActiveFilters

  const [internalPagination, setInternalPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: tableState.pageSize
  })

  const pagination: PaginationState = useMemo(
    () =>
      isServerSidePagination ? { pageIndex: tableState.page - 1, pageSize: tableState.pageSize } : internalPagination,
    [isServerSidePagination, tableState.page, tableState.pageSize, internalPagination]
  )

  useEffect(() => {
    if (!isServerSidePagination) {
      setInternalPagination((prev) => ({ ...prev, pageIndex: 0 }))
    }
  }, [isServerSidePagination])

  const setPagination = useCallback(
    (updater: PaginationState | ((old: PaginationState) => PaginationState)) => {
      const newPagination = typeof updater === 'function' ? updater(pagination) : updater

      if (isServerSidePagination) {
        setTableState({ page: newPagination.pageIndex + 1, pageSize: newPagination.pageSize })
      } else {
        setInternalPagination(newPagination)
      }
    },
    [pagination, isServerSidePagination, setTableState]
  )

  const columnResizeMode: ColumnResizeMode = 'onChange'

  const table = useReactTable({
    columns,
    data,
    getRowId,
    ...(isServerSidePagination ? { manualPagination: true, rowCount: totalCount } : { manualPagination: false }),
    manualFiltering: false,
    enableSortingRemoval: false,
    enableMultiSort,
    maxMultiSortColCount: enableMultiSort ? 3 : 1,
    enableColumnResizing,
    columnResizeMode,
    getCoreRowModel: getCoreRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    onRowSelectionChange: handleRowSelectionChange,
    onSortingChange: setSorting,
    state: { columnFilters, columnVisibility, pagination, rowSelection, sorting }
  })

  const { handleKeyDown, handleTableFocus } = useKeyboardNavigation({
    table,
    focusedCell,
    setFocusedCell,
    onRowClick
  })

  const isAllSelected = table.getIsAllRowsSelected()
  const isSomeSelected = table.getIsSomeRowsSelected()

  const handleSelectAll = useCallback(() => {
    if (isAllSelected || isSomeSelected) {
      table.resetRowSelection()
      setAnnouncement('All rows deselected')
    } else {
      table.toggleAllRowsSelected(true)
      setAnnouncement(`All ${table.getRowModel().rows.length} rows selected`)
    }
  }, [table, isAllSelected, isSomeSelected])

  const displayTotalCount = isServerSidePagination ? totalCount : table.getFilteredRowModel().rows.length
  const visibleColumns = table.getVisibleLeafColumns()
  const loadingRows = useMemo(
    () =>
      Array.from({ length: pagination.pageSize }, (_, slot) => ({
        id: `${id}-loading-${slot}`,
        isEven: slot % 2 === 0
      })),
    [id, pagination.pageSize]
  )

  const rowVirtualizer = useVirtualizer({
    count: table.getRowModel().rows.length,
    estimateSize: () => (tableState.density === 'compact' ? 40 : 52),
    getScrollElement: () => tableContainerRef.current,
    overscan: 10,
    enabled: enableVirtualization
  })

  const handleClearFilters = useCallback(() => {
    table.resetColumnFilters()
    setTableState({ search: null, page: 1 })
    onClearFilters?.()
  }, [table, setTableState, onClearFilters])

  const handleRowClick = useCallback(
    (row: Row<TData>) => {
      if (onRowClick) {
        onRowClick(row.original)
      }
    },
    [onRowClick]
  )

  return (
    <div className='space-y-4'>
      {/* Toolbar */}
      <DataTableToolbar
        table={table}
        searchColumnId={searchColumnId}
        searchPlaceholder={searchPlaceholder}
        enableViewOptions={true}
        enableDensityToggle={enableDensityToggle}
        density={tableState.density as DensityValue}
        onDensityChange={handleDensityChange}
        filterSlot={filterComponents}
        actionSlot={
          <div className='flex items-center gap-2'>
            {!readOnly && (
              <Button variant='outline' size='sm' onClick={handleSelectAll} className='h-9 gap-2'>
                {isAllSelected || isSomeSelected ? (
                  <>
                    <CircleXIcon className='h-4 w-4' />
                    Deselect All
                  </>
                ) : (
                  <>
                    <CheckIcon className='h-4 w-4' />
                    Select All
                  </>
                )}
              </Button>
            )}

            {toolbarActions}
          </div>
        }
      />

      {/* Table */}
      <div className='rounded-lg border bg-background overflow-hidden'>
        <div
          className={cn('overflow-x-hidden', enableVirtualization && 'max-h-[600px] overflow-y-auto')}
          ref={tableContainerRef}
        >
          <table
            aria-label={`${entityName} data table`}
            aria-rowcount={displayTotalCount}
            aria-busy={isLoading}
            className='w-full table-fixed outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'
            onKeyDown={handleKeyDown}
            onFocus={handleTableFocus}
          >
            <thead className='sticky top-0 z-10 bg-muted border-b'>
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    const canSort = header.column.getCanSort()
                    const sorted = header.column.getIsSorted()
                    const columnId = header.column.id.toLowerCase()
                    const isNumeric =
                      columnId === 'id' ||
                      ['amount', 'price', 'count', 'total', 'quantity', 'balance'].some((key) => columnId.includes(key))
                    const ariaSort = canSort
                      ? sorted === 'asc'
                        ? 'ascending'
                        : sorted === 'desc'
                          ? 'descending'
                          : 'none'
                      : undefined

                    return (
                      <th
                        key={header.id}
                        colSpan={header.colSpan}
                        aria-sort={ariaSort}
                        scope='col'
                        className={cn(
                          'h-11 px-3 font-semibold text-xs uppercase tracking-wider text-muted-foreground relative',
                          'transition-colors duration-150',
                          isNumeric ? 'text-right' : 'text-left',
                          canSort && 'cursor-pointer select-none hover:bg-muted/60 hover:text-foreground',
                          sorted && 'text-foreground bg-muted/40'
                        )}
                        style={{ width: header.getSize() }}
                        onClick={canSort ? header.column.getToggleSortingHandler() : undefined}
                        onKeyDown={
                          canSort
                            ? (event) => {
                                if (event.key === 'Enter' || event.key === ' ') {
                                  event.preventDefault()
                                  header.column.getToggleSortingHandler()?.(event)
                                }
                              }
                            : undefined
                        }
                        tabIndex={canSort ? 0 : -1}
                      >
                        {header.isPlaceholder ? null : (
                          <div className={cn('flex items-center gap-1.5', isNumeric ? 'justify-end' : 'justify-start')}>
                            <span className='truncate'>
                              {flexRender(header.column.columnDef.header, header.getContext())}
                            </span>
                            {canSort && (
                              <span
                                className={cn(
                                  'transition-all duration-200 flex-shrink-0',
                                  sorted ? 'opacity-100' : 'opacity-40'
                                )}
                              >
                                {sorted === 'asc' && <ArrowUpIcon className='h-3.5 w-3.5' />}
                                {sorted === 'desc' && <ArrowDownIcon className='h-3.5 w-3.5' />}
                                {!sorted && <ChevronsUpDownIcon className='h-3.5 w-3.5' />}
                              </span>
                            )}
                          </div>
                        )}
                        {header.column.getCanResize() && (
                          <hr
                            onMouseDown={header.getResizeHandler()}
                            onTouchStart={header.getResizeHandler()}
                            onClick={(e) => e.stopPropagation()}
                            onDoubleClick={() => header.column.resetSize()}
                            onKeyDown={(event) => {
                              event.stopPropagation()
                              if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
                              event.preventDefault()
                              const step = event.shiftKey ? 50 : 10
                              const width = header.column.getSize() + (event.key === 'ArrowLeft' ? -step : step)
                              const minSize = header.column.columnDef.minSize ?? 20
                              const maxSize = header.column.columnDef.maxSize ?? Number.MAX_SAFE_INTEGER
                              table.setColumnSizing((sizes) => ({
                                ...sizes,
                                [header.column.id]: Math.min(maxSize, Math.max(minSize, width))
                              }))
                            }}
                            tabIndex={0}
                            aria-label={`Resize ${header.column.id} column`}
                            aria-orientation='vertical'
                            aria-valuenow={header.column.getSize()}
                            aria-valuemin={header.column.columnDef.minSize ?? 20}
                            aria-valuemax={header.column.columnDef.maxSize ?? Number.MAX_SAFE_INTEGER}
                            className={cn(
                              'absolute right-0 top-0 m-0 h-full w-1 border-0 cursor-col-resize select-none touch-none',
                              'hover:bg-primary/50 focus-visible:bg-primary focus-visible:outline-none transition-colors',
                              header.column.getIsResizing() && 'bg-primary'
                            )}
                          />
                        )}
                      </th>
                    )
                  })}
                </tr>
              ))}
            </thead>

            <tbody className='divide-y divide-border'>
              {isLoading ? (
                loadingRows.map((loadingRow) => (
                  <tr
                    key={loadingRow.id}
                    className={cn(
                      loadingRow.isEven ? 'bg-background' : 'bg-muted/30',
                      tableState.density === 'compact'
                        ? 'h-10'
                        : tableState.density === 'spacious'
                          ? 'h-[72px]'
                          : 'h-[52px]'
                    )}
                  >
                    {visibleColumns.map((column, colIndex) => (
                      <td
                        key={column.id}
                        className={cn(
                          'px-3 align-middle',
                          tableState.density === 'compact'
                            ? 'py-1'
                            : tableState.density === 'spacious'
                              ? 'py-4'
                              : 'py-2'
                        )}
                      >
                        <Skeleton
                          className={cn(
                            'h-4',
                            colIndex === 0
                              ? 'w-4'
                              : colIndex === visibleColumns.length - 1
                                ? 'w-8'
                                : 'w-full max-w-[200px]'
                          )}
                        />
                      </td>
                    ))}
                  </tr>
                ))
              ) : table.getRowModel().rows?.length ? (
                enableVirtualization ? (
                  <>
                    {rowVirtualizer.getVirtualItems().length > 0 && (
                      <tr style={{ height: `${rowVirtualizer.getVirtualItems()[0]?.start ?? 0}px` }} />
                    )}
                    {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                      const row = table.getRowModel().rows[virtualRow.index]
                      return (
                        <TableRow
                          key={row.id}
                          row={row}
                          rowIndex={virtualRow.index}
                          density={tableState.density}
                          onClick={handleRowClick}
                          isClickable={!!onRowClick}
                          isActive={activeRowId === String(row.original.id)}
                          virtualRef={rowVirtualizer.measureElement}
                          dataIndex={virtualRow.index}
                          focusedCell={focusedCell}
                        />
                      )
                    })}
                    {rowVirtualizer.getVirtualItems().length > 0 && (
                      <tr
                        style={{
                          height: `${rowVirtualizer.getTotalSize() - (rowVirtualizer.getVirtualItems()[rowVirtualizer.getVirtualItems().length - 1]?.end ?? 0)}px`
                        }}
                      />
                    )}
                  </>
                ) : (
                  table
                    .getRowModel()
                    .rows.map((row, rowIndex) => (
                      <TableRow
                        key={row.id}
                        row={row}
                        rowIndex={rowIndex}
                        density={tableState.density}
                        onClick={handleRowClick}
                        isClickable={!!onRowClick}
                        isActive={activeRowId === String(row.original.id)}
                        focusedCell={focusedCell}
                      />
                    ))
                )
              ) : (
                <tr>
                  <td className='h-40 text-center' colSpan={columns.length}>
                    {emptyState || (
                      <div className='flex flex-col items-center justify-center gap-3 py-8'>
                        <SearchXIcon className='h-10 w-10 text-muted-foreground/50' />
                        <div>
                          <p className='font-medium'>No {entityName} found</p>
                          {hasActiveFilters && (
                            <p className='text-sm text-muted-foreground mt-1'>
                              Try adjusting your search or filter criteria
                            </p>
                          )}
                        </div>
                        {hasActiveFilters && (
                          <Button variant='outline' size='sm' onClick={handleClearFilters}>
                            <CircleXIcon className='mr-2 h-4 w-4' />
                            Clear all filters
                          </Button>
                        )}
                      </div>
                    )}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <DataTablePagination
        table={table}
        displayTotalCount={displayTotalCount}
        pageSizeOptions={pageSizeOptions}
        id={id}
      />

      <div role='status' aria-live='polite' aria-atomic='true' className='sr-only'>
        {announcement}
      </div>

      <AlertDialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
        <AlertDialogContent style={{ overscrollBehavior: 'contain' }}>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {pendingDeleteIds.length === 1 ? 'Item' : `${pendingDeleteIds.length} Items`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete{' '}
              {pendingDeleteIds.length === 1 ? 'this item' : `these ${pendingDeleteIds.length} items`}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                onDeleteSelected?.(pendingDeleteIds)
                setDeleteConfirmOpen(false)
                setPendingDeleteIds([])
              }}
              disabled={isDeleting}
              className={buttonVariants({ variant: 'destructive' })}
            >
              {isDeleting && <Loader2Icon className='mr-2 h-4 w-4 animate-spin' aria-hidden='true' />}
              {isDeleting ? 'Deleting…' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export type { ColumnDef, RowSelectionState, TanstackTable }
