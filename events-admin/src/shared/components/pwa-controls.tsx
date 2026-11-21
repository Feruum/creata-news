import { Download, RefreshCw, X } from 'lucide-react'
import type { ReactNode } from 'react'
import { usePwa, usePwaController } from '@/shared/hooks/use-pwa'
import { PwaContext } from '@/shared/lib/pwa-context'
import { Button } from '@/shared/ui/components/button'

export function PwaProvider({ children }: { children: ReactNode }) {
  const state = usePwaController()
  return (
    <PwaContext.Provider value={state}>
      {children}
      <PwaControls />
    </PwaContext.Provider>
  )
}

export function PwaControls() {
  const { canInstall, hasUpdate, dismissed, install, applyUpdate, dismiss } = usePwa()
  if ((!canInstall && !hasUpdate) || dismissed) return null
  return (
    <div
      role='status'
      aria-label='App installation and updates'
      className='fixed bottom-4 left-1/2 z-[70] flex max-w-[calc(100%-2rem)] -translate-x-1/2 items-center gap-3 rounded-lg border bg-background p-3 shadow-md'
    >
      <p className='text-sm'>{hasUpdate ? 'An app update is ready.' : 'Install AITU EMS for quick access.'}</p>
      <Button size='sm' onClick={hasUpdate ? applyUpdate : () => void install()}>
        {hasUpdate ? <RefreshCw className='size-4' /> : <Download className='size-4' />}
        {hasUpdate ? 'Update' : 'Install'}
      </Button>
      <Button size='icon' variant='ghost' aria-label='Dismiss app prompt' onClick={dismiss}>
        <X />
      </Button>
    </div>
  )
}
