'use client'

import { useContext } from 'react'
import { DragDropContext } from './dnd-context'

export function useDragDrop() {
  const context = useContext(DragDropContext)
  if (!context) {
    throw new Error('useDragDrop must be used within a DragDropProvider')
  }
  return context
}
