'use client'

import { useRender } from '@base-ui/react/use-render'
import * as React from 'react'

/** Base UI composition for custom wrappers that forward props onto their child. */
export const Slot = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(({ children, ...props }, ref) => {
  const child = React.Children.only(children)
  if (!React.isValidElement(child)) throw new Error('Slot requires one React element')
  return useRender({ render: child, props, ref })
})
Slot.displayName = 'Slot'
