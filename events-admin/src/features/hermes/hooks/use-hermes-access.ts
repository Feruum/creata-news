import { useCallback, useEffect, useRef, useState } from 'react'
import { useAuth } from '@/shared/lib/auth'
import { browserIsOnline, onPrivateCacheInvalidated } from '@/shared/lib/offline-runtime'
import { HermesAccessError, type HermesAccessInfo, loadHermesAccess } from '../api/hermes-access'

type AccessState =
  | { status: 'loading' }
  | { status: 'ready'; info: HermesAccessInfo }
  | { status: 'denied' | 'unavailable' }

export function useHermesAccess() {
  const { session, isAuthenticated } = useAuth()
  const expiresAt = Date.parse(session?.expires_at ?? '')
  const owner =
    isAuthenticated &&
    session?.active &&
    session.id &&
    session.identity?.id &&
    Number.isFinite(expiresAt) &&
    expiresAt > Date.now() &&
    browserIsOnline()
      ? JSON.stringify([session.id, session.identity.id])
      : null
  const [stored, setStored] = useState<{ owner: string | null; state: AccessState }>({
    owner: null,
    state: { status: 'loading' }
  })
  const lifecycle = useRef<{ owner: string | null; generation: number; controller: AbortController | null }>({
    owner: null,
    generation: 0,
    controller: null
  })

  const run = useCallback((requestOwner: string, expiry: number) => {
    lifecycle.current.controller?.abort()
    const controller = new AbortController()
    const generation = ++lifecycle.current.generation
    lifecycle.current.owner = requestOwner
    lifecycle.current.controller = controller
    setStored({ owner: requestOwner, state: { status: 'loading' } })
    const current = () =>
      lifecycle.current.owner === requestOwner &&
      lifecycle.current.generation === generation &&
      !controller.signal.aborted &&
      browserIsOnline() &&
      expiry > Date.now()

    void loadHermesAccess(controller.signal)
      .then((info) => {
        if (current()) setStored({ owner: requestOwner, state: { status: 'ready', info } })
      })
      .catch((error: unknown) => {
        if (current())
          setStored({
            owner: requestOwner,
            state: { status: error instanceof HermesAccessError ? error.reason : 'unavailable' }
          })
      })
  }, [])

  const reload = useCallback(() => {
    if (owner && expiresAt > Date.now() && browserIsOnline()) run(owner, expiresAt)
  }, [owner, expiresAt, run])

  useEffect(() => {
    if (!owner) return
    run(owner, expiresAt)
    const invalidate = () => {
      lifecycle.current.generation += 1
      lifecycle.current.owner = null
      lifecycle.current.controller?.abort()
      setStored({ owner, state: { status: 'unavailable' } })
    }
    let timer: ReturnType<typeof setTimeout>
    const expire = () => {
      const remaining = expiresAt - Date.now()
      if (remaining <= 0) invalidate()
      else timer = setTimeout(expire, Math.min(remaining, 2_147_483_647))
    }
    expire()
    const unsubscribe = onPrivateCacheInvalidated(invalidate)
    window.addEventListener('offline', invalidate)
    window.addEventListener('focus', reload)
    return () => {
      lifecycle.current.generation += 1
      lifecycle.current.owner = null
      lifecycle.current.controller?.abort()
      clearTimeout(timer)
      unsubscribe()
      window.removeEventListener('offline', invalidate)
      window.removeEventListener('focus', reload)
    }
  }, [owner, expiresAt, reload, run])

  const state: AccessState =
    owner && stored.owner === owner ? stored.state : owner ? { status: 'loading' } : { status: 'unavailable' }

  return { state, reload }
}
