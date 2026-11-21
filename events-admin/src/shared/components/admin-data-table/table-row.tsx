import { flexRender, type Row } from '@tanstack/react-table'

import { cn } from '@/shared/ui/lib/utils'

export interface TableRowProps<TData> {
  row: Row<TData>
  rowIndex: number
  density: string
  onClick?: (row: Row<TData>) => void
  isClickable: boolean
  isActive?: boolean
  virtualRef?: (element: HTMLTableRowElement | null) => void
  dataIndex?: number
  focusedCell?: { rowIndex: number; colIndex: number } | null
}

export function TableRow<TData>({
  row,
  rowIndex,
  density,
  onClick,
  isClickable,
  isActive = false,
  virtualRef,
  dataIndex,
  focusedCell
}: TableRowProps<TData>) {
  const isEven = rowIndex % 2 === 0

  return (
    <tr
      ref={virtualRef}
      aria-rowindex={rowIndex + 2}
      aria-selected={row.getIsSelected()}
      data-index={dataIndex}
      data-state={row.getIsSelected() && 'selected'}
      data-preview-active={isActive || undefined}
      tabIndex={isClickable ? -1 : undefined}
      className={cn(
        'transition-colors duration-150 border-l-2 border-l-transparent',
        isEven ? 'bg-background' : 'bg-muted/30',
        'hover:bg-muted/50 hover:border-l-primary/50',
        'data-[state=selected]:bg-primary/8 data-[state=selected]:border-l-primary data-[state=selected]:hover:bg-primary/12',
        isClickable && 'cursor-pointer',
        isActive && 'bg-muted/70! border-l-foreground/20!',
        density === 'compact' ? 'h-10' : density === 'spacious' ? 'h-[72px]' : 'h-[52px]'
      )}
      onClick={(event) => {
        // Portaled dialogs and select popups still bubble through the React row.
        if (!(event.target instanceof Node) || !event.currentTarget.contains(event.target)) return
        if (isClickable) event.currentTarget.focus({ preventScroll: true })
        onClick?.(row)
      }}
    >
      {row.getVisibleCells().map((cell, colIndex) => {
        const columnId = cell.column.id.toLowerCase()
        const isNumeric =
          columnId === 'id' ||
          ['amount', 'price', 'count', 'total', 'quantity', 'balance'].some((key) => columnId.includes(key))
        const isFocused = focusedCell?.rowIndex === rowIndex && focusedCell?.colIndex === colIndex

        return (
          <td
            key={cell.id}
            id={`cell-${rowIndex}-${colIndex}`}
            tabIndex={isFocused ? 0 : -1}
            className={cn(
              'px-3 align-middle overflow-hidden outline-none',
              'focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
              density === 'compact' ? 'py-1' : density === 'spacious' ? 'py-4' : 'py-2',
              isNumeric ? 'text-right tabular-nums' : 'text-left'
            )}
            style={{ width: cell.column.getSize() }}
          >
            <div className={cn('truncate', isNumeric ? 'text-right' : 'text-left')}>
              {flexRender(cell.column.columnDef.cell, cell.getContext())}
            </div>
          </td>
        )
      })}
    </tr>
  )
}
