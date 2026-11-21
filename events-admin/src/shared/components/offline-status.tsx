import { WifiOff } from 'lucide-react'
import { useEffect } from 'react'
import { useOfflineState } from '@/shared/hooks/use-offline-state'
import { trackProductEvent } from '@/shared/lib/telemetry'
import { useAuth } from '@/shared/lib/auth'
import { Button } from '@/shared/ui/components/button'

export function OfflineStatus() {
  const state = useOfflineState()
  const { checkSession } = useAuth()
  useEffect(() => {
    if (state.mode === 'offline-readonly') {
      trackProductEvent('offline_open', { surface: 'admin', source: 'browser', outcome: 'success' })
    }
  }, [state.mode])

  if (state.mode !== 'offline-readonly' && state.storageAvailable) return null
  return (
    <div
      role='status'
      aria-live='polite'
      className='flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm'
    >
      <WifiOff className='size-4 shrink-0' aria-hidden='true' />
      <span>
        {state.mode === 'offline-readonly'
          ? 'Offline · Read-only. Saved views may be out of date. Reconnect to make changes.'
          : 'Offline storage is unavailable. This browser cannot save views for offline use.'}
      </span>
      {state.mode === 'offline-readonly' && (
        <span className='text-xs text-muted-foreground'>
          Last updated: {new Date(state.lastUpdatedAt ?? state.identity?.verifiedAt ?? Date.now()).toLocaleString()}
        </span>
      )}
      {state.mode === 'offline-readonly' && (
        <Button size='sm' variant='outline' onClick={() => void checkSession()}>
          Retry connection
        </Button>
      )}
    </div>
  )
}
