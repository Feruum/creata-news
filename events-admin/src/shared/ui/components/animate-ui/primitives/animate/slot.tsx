'use client'

import { type HTMLMotionProps, motion } from 'motion/react'
import * as React from 'react'
import { Slot as BaseSlot } from '../../../slot'

type WithAsChild<T> = T & { asChild?: boolean }

type SlotProps = HTMLMotionProps<'span'> & {
  children: React.ReactNode
}

const Slot = React.forwardRef<HTMLSpanElement, SlotProps>(({ children, ...props }, ref) => {
  const child = React.Children.only(children) as React.ReactElement
  const MotionSlot = motion.create(BaseSlot)

  return (
    <MotionSlot ref={ref} {...props}>
      {child}
    </MotionSlot>
  )
})

Slot.displayName = 'Slot'

export { Slot, type SlotProps, type WithAsChild }
