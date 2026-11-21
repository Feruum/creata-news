import { createContext } from 'react'

export interface PwaState {
  canInstall: boolean
  isIos: boolean
  isInstalled: boolean
  hasUpdate: boolean
  dismissed: boolean
  install: (surface?: 'sidebar' | 'admin') => Promise<void>
  applyUpdate: () => void
  dismiss: () => void
}

export const PwaContext = createContext<PwaState | null>(null)
