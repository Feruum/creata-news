import { useContext, useEffect, useRef, useState } from 'react'
import { PwaContext, type PwaState } from '@/shared/lib/pwa-context'
import { trackProductEvent } from '@/shared/lib/telemetry'

interface InstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

export function usePwa(): PwaState {
  const state = useContext(PwaContext)
  if (!state) throw new Error('usePwa requires a PwaProvider')
  return state
}

export function usePwaController(): PwaState {
  const [prompt, setPrompt] = useState<InstallPromptEvent | null>(null)
  const [waiting, setWaiting] = useState<ServiceWorker | null>(null)
  const [dismissed, setDismissed] = useState(false)
  const [isInstalled, setInstalled] = useState(
    () =>
      window.matchMedia('(display-mode: standalone)').matches ||
      ('standalone' in navigator && navigator.standalone === true)
  )
  const applyingUpdate = useRef(false)
  const isIos =
    /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1)

  useEffect(() => {
    const installPrompt = (event: Event) => {
      event.preventDefault()
      setPrompt(event as InstallPromptEvent)
      setDismissed(false)
    }
    const installed = () => {
      setPrompt(null)
      setInstalled(true)
      trackProductEvent('pwa_install', { surface: 'admin', source: 'browser', outcome: 'success' })
    }
    window.addEventListener('beforeinstallprompt', installPrompt)
    window.addEventListener('appinstalled', installed)
    const controllerChange = () => {
      if (applyingUpdate.current) window.location.reload()
    }
    let stopped = false
    if (import.meta.env.PROD && 'serviceWorker' in navigator) {
      navigator.serviceWorker.addEventListener('controllerchange', controllerChange)
      void navigator.serviceWorker
        .register(`${import.meta.env.BASE_URL}sw.js`, {
          scope: import.meta.env.BASE_URL,
          updateViaCache: 'none'
        })
        .then((registration) => {
          if (stopped) return
          if (registration.waiting) setWaiting(registration.waiting)
          registration.addEventListener('updatefound', () => {
            const installing = registration.installing
            installing?.addEventListener('statechange', () => {
              if (!stopped && installing.state === 'installed' && navigator.serviceWorker.controller) {
                setWaiting(registration.waiting)
                setDismissed(false)
              }
            })
          })
        })
        .catch(() => {
          /* Browsers without SW support can continue online. */
        })
    }
    return () => {
      stopped = true
      navigator.serviceWorker?.removeEventListener('controllerchange', controllerChange)
      window.removeEventListener('beforeinstallprompt', installPrompt)
      window.removeEventListener('appinstalled', installed)
    }
  }, [])

  return {
    canInstall: !!prompt && !isInstalled,
    isIos,
    isInstalled,
    hasUpdate: !!waiting,
    dismissed,
    install: async (surface = 'admin') => {
      if (!prompt) return
      trackProductEvent('pwa_install', { surface, source: 'browser', outcome: 'started' })
      try {
        await prompt.prompt()
        await prompt.userChoice
      } catch {
        trackProductEvent('pwa_install', { surface, source: 'browser', outcome: 'error' })
      } finally {
        setPrompt(null)
      }
    },
    applyUpdate: () => {
      applyingUpdate.current = true
      waiting?.postMessage({ type: 'SKIP_WAITING' })
    },
    dismiss: () => setDismissed(true)
  }
}
