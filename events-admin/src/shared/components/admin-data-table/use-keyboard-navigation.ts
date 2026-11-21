import type { Table as TanstackTable } from '@tanstack/react-table'
import { type FocusEvent, type KeyboardEvent, useCallback } from 'react'

type FocusedCell = { rowIndex: number; colIndex: number } | null

type UseKeyboardNavigationParams<TData> = {
  table: TanstackTable<TData>
  focusedCell: FocusedCell
  setFocusedCell: (cell: FocusedCell) => void
  onRowClick?: (row: TData) => void
}

export function useKeyboardNavigation<TData>({
  table,
  focusedCell,
  setFocusedCell,
  onRowClick
}: UseKeyboardNavigationParams<TData>) {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTableElement>) => {
      if (event.defaultPrevented) return
      if (
        event.target instanceof Element &&
        event.target.closest(
          'button, a, input, select, textarea, [contenteditable="true"], [role="checkbox"], [role="switch"], [role="combobox"], [role="textbox"]'
        )
      )
        return
      const rows = table.getRowModel().rows
      const visibleColumns = table.getVisibleFlatColumns()
      if (!focusedCell || rows.length === 0) return

      const { rowIndex, colIndex } = focusedCell
      let newRowIndex = rowIndex
      let newColIndex = colIndex
      let handled = false

      switch (event.key) {
        case 'ArrowUp':
          if (rowIndex > 0) {
            newRowIndex = rowIndex - 1
            handled = true
          }
          break
        case 'ArrowDown':
          if (rowIndex < rows.length - 1) {
            newRowIndex = rowIndex + 1
            handled = true
          }
          break
        case 'ArrowLeft':
          if (colIndex > 0) {
            newColIndex = colIndex - 1
            handled = true
          }
          break
        case 'ArrowRight':
          if (colIndex < visibleColumns.length - 1) {
            newColIndex = colIndex + 1
            handled = true
          }
          break
        case 'Home':
          if (event.ctrlKey) {
            newRowIndex = 0
            newColIndex = 0
          } else {
            newColIndex = 0
          }
          handled = true
          break
        case 'End':
          if (event.ctrlKey) {
            newRowIndex = rows.length - 1
            newColIndex = visibleColumns.length - 1
          } else {
            newColIndex = visibleColumns.length - 1
          }
          handled = true
          break
        case 'Enter':
        case ' ':
          if (onRowClick) {
            onRowClick(rows[rowIndex].original)
            handled = true
          }
          break
        case 'Escape':
          setFocusedCell(null)
          event.currentTarget.blur()
          handled = true
          break
      }

      if (handled) {
        event.preventDefault()
        if (newRowIndex !== rowIndex || newColIndex !== colIndex) {
          setFocusedCell({ rowIndex: newRowIndex, colIndex: newColIndex })
          const cellId = `cell-${newRowIndex}-${newColIndex}`
          document.getElementById(cellId)?.focus()
        }
      }
    },
    [focusedCell, onRowClick, setFocusedCell, table]
  )

  const handleTableFocus = useCallback(
    (event: FocusEvent<HTMLElement>) => {
      const target = event.target
      if (target instanceof Element) {
        const row = target.closest('tr[aria-rowindex]')
        const rowIndex = Number(row?.getAttribute('aria-rowindex')) - 2
        const cell = target.closest('td')
        if (row && rowIndex >= 0 && rowIndex < table.getRowModel().rows.length) {
          setFocusedCell({ rowIndex, colIndex: cell instanceof HTMLTableCellElement ? cell.cellIndex : 0 })
          return
        }
      }
      if (!focusedCell && table.getRowModel().rows.length > 0) {
        setFocusedCell({ rowIndex: 0, colIndex: 0 })
      }
    },
    [focusedCell, setFocusedCell, table]
  )

  return { handleKeyDown, handleTableFocus }
}
