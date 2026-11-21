import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger
} from '@/shared/ui/components/drawer'
import { useMediaQuery } from '@/shared/ui/hooks/use-media-query'
import { cn } from '@/shared/ui/lib/utils'

/**
 * SchrodingersModal - It's both a Drawer AND a Dialog until you observe the screen size! 🐱📦
 *
 * A quantum superposition of UI components that collapses into either:
 * - A majestic Dialog on desktop (because big screens deserve big modals)
 * - A sleek Drawer on mobile (because thumbs need room to swipe)
 *
 * Named after Schrödinger's famous thought experiment, where a cat is simultaneously
 * alive and dead until observed. Similarly, this component is simultaneously a drawer
 * and a dialog until the viewport width is measured! 🎭
 */

interface SchrodingersModalProps {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  trigger?: React.ReactElement
  title?: string
  description?: string
  children: React.ReactNode
  footer?: React.ReactNode
  className?: string
}

export function SchrodingersModal({
  open,
  onOpenChange,
  trigger,
  title,
  description,
  children,
  footer,
  className
}: SchrodingersModalProps) {
  const [isOpen, setIsOpen] = useState(false)
  const isDesktop = useMediaQuery('(min-width: 768px)')

  const handleOpenChange = (newOpen: boolean) => {
    setIsOpen(newOpen)
    onOpenChange?.(newOpen)
  }

  const currentOpen = open !== undefined ? open : isOpen

  // Desktop: The Dialog Form (for sophisticated users with large screens)
  if (isDesktop) {
    return (
      <Dialog onOpenChange={handleOpenChange} open={currentOpen}>
        {trigger && <DialogTrigger render={trigger}></DialogTrigger>}
        <DialogContent className={cn('sm:max-w-[425px]', className)}>
          <DialogHeader>
            {title && <DialogTitle>{title}</DialogTitle>}
            {description && <DialogDescription>{description}</DialogDescription>}
          </DialogHeader>
          {children}
        </DialogContent>
      </Dialog>
    )
  }

  // Mobile: The Drawer Form (for thumb-wielding warriors)
  return (
    <Drawer onOpenChange={handleOpenChange} open={currentOpen}>
      {trigger && <DrawerTrigger render={trigger}></DrawerTrigger>}
      <DrawerContent>
        <DrawerHeader className='text-left'>
          {title && <DrawerTitle>{title}</DrawerTitle>}
          {description && <DrawerDescription>{description}</DrawerDescription>}
        </DrawerHeader>
        <div className={cn('min-h-0 flex-1 overflow-y-auto px-4', className)}>{children}</div>
        {footer && <DrawerFooter className='pt-2'>{footer}</DrawerFooter>}
      </DrawerContent>
    </Drawer>
  )
}
