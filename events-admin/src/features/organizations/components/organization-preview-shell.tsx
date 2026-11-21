import { Dialog } from '@base-ui/react/dialog'
import { Resizable } from 're-resizable'
import { type ReactNode, type RefObject, useEffect, useRef, useState } from 'react'
import type { OrganizationPreviewLayout } from '../types'
import './organization-preview.css'

interface OrganizationPreviewShellProps {
  open: boolean
  layout: OrganizationPreviewLayout
  onClose: () => void
  returnFocusRef: RefObject<HTMLElement | null>
  children: ReactNode
}

export function OrganizationPreviewShell({
  open,
  layout,
  onClose,
  returnFocusRef,
  children
}: OrganizationPreviewShellProps) {
  const [viewport, setViewport] = useState(() => ({ width: window.innerWidth, height: window.innerHeight }))
  const [drawerWidth, setDrawerWidth] = useState(668)
  const [popupSize, setPopupSize] = useState({ width: 900, height: 820 })
  const popupRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const resize = () => setViewport({ width: window.innerWidth, height: window.innerHeight })
    window.addEventListener('resize', resize)
    return () => window.removeEventListener('resize', resize)
  }, [])

  const compact = viewport.width < 1024
  const minimized = layout === 'minimize'
  const full = !minimized && (compact || layout === 'full-page')
  const popup = !compact && layout === 'pop-up'
  const maxWidth = Math.max(480, viewport.width - 48)
  const width = minimized
    ? Math.min(340, viewport.width - 24)
    : full
      ? viewport.width
      : Math.min(popup ? popupSize.width : drawerWidth, maxWidth)
  const height = minimized ? 48 : popup ? Math.min(popupSize.height, viewport.height - 48) : viewport.height
  const resizeWidth = (next: number) => setDrawerWidth(Math.min(maxWidth, Math.max(480, next)))

  return (
    <Dialog.Root
      open={open}
      modal={compact && !minimized}
      disablePointerDismissal
      onOpenChange={(next) => !next && onClose()}
    >
      <Dialog.Portal>
        <Resizable
          className='club-preview-frame'
          size={{ width, height }}
          minWidth={minimized ? Math.min(280, viewport.width - 24) : full ? viewport.width : 480}
          maxWidth={full ? viewport.width : maxWidth}
          minHeight={minimized ? 48 : Math.min(360, viewport.height)}
          maxHeight={popup ? viewport.height - 48 : viewport.height}
          enable={full || minimized ? false : popup ? { right: true, bottom: true, bottomRight: true } : { left: true }}
          handleStyles={{ left: { left: -4, width: 8, zIndex: 1 } }}
          handleComponent={{
            left: (
              <hr
                tabIndex={0}
                aria-label='Resize club preview'
                aria-orientation='vertical'
                aria-valuenow={Math.round(width)}
                aria-valuemin={480}
                aria-valuemax={Math.round(maxWidth)}
                className='club-preview-resizer'
                onKeyDown={(event) => {
                  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
                  event.preventDefault()
                  resizeWidth(
                    event.key === 'Home'
                      ? 480
                      : event.key === 'End'
                        ? maxWidth
                        : width + (event.key === 'ArrowLeft' ? 24 : -24)
                  )
                }}
              />
            )
          }}
          onResize={(_, __, element) => {
            if (popup) setPopupSize({ width: element.offsetWidth, height: element.offsetHeight })
            else setDrawerWidth(element.offsetWidth)
          }}
          style={{
            position: 'fixed',
            zIndex: 50,
            ...(minimized
              ? { bottom: 12, right: 12 }
              : popup
                ? { top: '50%', left: '50%', transform: 'translate(-50%, -50%)' }
                : { top: 0, right: 0 })
          }}
        >
          <Dialog.Popup
            ref={popupRef}
            initialFocus={popupRef}
            finalFocus={() => {
              const target = returnFocusRef.current
              if (!target?.isConnected) return true
              // A row may contain checkboxes; restore the row itself, not its first tabbable child.
              queueMicrotask(() => target.isConnected && target.focus({ preventScroll: true }))
              return false
            }}
            className='club-preview'
            data-testid='club-preview'
            data-layout={layout}
          >
            <Dialog.Title className='sr-only'>Club preview</Dialog.Title>
            <Dialog.Description className='sr-only'>
              Club details, team, role structure and membership history.
            </Dialog.Description>
            {children}
          </Dialog.Popup>
        </Resizable>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
