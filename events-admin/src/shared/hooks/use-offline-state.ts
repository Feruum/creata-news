import { useSyncExternalStore } from 'react'
import { getOfflineState, subscribeOffline } from '@/shared/lib/offline-runtime'

export function useOfflineState() {
  return useSyncExternalStore(subscribeOffline, getOfflineState, getOfflineState)
}
