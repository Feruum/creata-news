import type { Session, UiNodeInputAttributes } from '@ory/client'
import * as React from 'react'
import { useOfflineState } from '@/shared/hooks/use-offline-state'
import {
  canResumeAccount,
  captureAccountIntent,
  currentAccountIntent,
  noteLocalLogout,
  resetLocalLogout
} from './account-revocation'
import { getUserDisplayName, getUserEmail } from './auth-utils'
import { readBrowserSetting, writeBrowserSetting } from './browser-storage'
import { kratos } from './kratos'
import {
  browserIsOnline,
  clearPrivateSnapshots,
  offlineSnapshots,
  onPrivateCacheInvalidated,
  restoreOfflineIdentity,
  setSessionVerifier,
  verifyOfflineIdentity
} from './offline-runtime'
import type { OfflineIdentity, SnapshotOwner } from './offline-snapshots'

interface KratosErrorWithResponse {
  response?: {
    status?: number
  }
}

export interface AuthContext {
  isAuthenticated: boolean
  isLoading: boolean
  isGuest: boolean
  isOfflineReadonly: boolean
  offlineIdentity: OfflineIdentity | null
  session: Session | null
  login: (returnTo?: string) => Promise<void>
  logout: () => Promise<void>
  checkSession: () => Promise<Session | null>
  continueAsGuest: () => void
  exitGuestMode: () => void
}

const AuthContext = React.createContext<AuthContext | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [session, setSession] = React.useState<Session | null>(null)
  const offline = useOfflineState()
  const checkGeneration = React.useRef(0)
  const checkSequence = React.useRef(0)
  const [isLoading, setIsLoading] = React.useState(true)
  const [isGuest, setIsGuest] = React.useState(() => {
    return readBrowserSetting('guestMode') === 'true'
  })

  const isAuthenticated = !!session?.active && offline.mode === 'online'
  const isOfflineReadonly = offline.mode === 'offline-readonly' && !!offline.identity

  const checkSession = React.useCallback(async (): Promise<Session | null> => {
    const sequence = ++checkSequence.current
    const generation = ++checkGeneration.current
    let requestOwner: SnapshotOwner | null = null
    setIsLoading(true)
    try {
      const intentCheckpoint = await captureAccountIntent()
      if (generation !== checkGeneration.current) return null
      if (!browserIsOnline()) {
        await restoreOfflineIdentity()
        if (generation !== checkGeneration.current) return null
        setSession(null)
        return null
      }
      requestOwner = await offlineSnapshots.captureOwner(true)
      if (generation !== checkGeneration.current) return null
      const { data } = await kratos.toSession()
      if (generation !== checkGeneration.current) return null
      if (data.identity?.id && !(await canResumeAccount(data.identity.id, intentCheckpoint))) {
        if (generation !== checkGeneration.current) return null
        await clearPrivateSnapshots('logout', undefined, requestOwner)
        return null
      }
      if (!data.active || !data.identity?.id) {
        await clearPrivateSnapshots('session-expired', undefined, requestOwner)
        return null
      }
      const verified = await verifyOfflineIdentity(
        {
          identityId: data.identity.id,
          displayName: getUserDisplayName(data),
          email: getUserEmail(data),
          verifiedAt: Date.now(),
          sessionExpiresAt: Date.parse(data.expires_at ?? '') || Number.POSITIVE_INFINITY
        },
        requestOwner,
        intentCheckpoint
      )
      if (!verified || generation !== checkGeneration.current) return null
      if (!(await canResumeAccount(data.identity.id, intentCheckpoint))) return null
      if (generation !== checkGeneration.current) return null
      setSession(data)
      if (data.active) {
        setIsGuest(false)
        writeBrowserSetting('guestMode', null)
      }
      return data
    } catch (error: unknown) {
      if (generation !== checkGeneration.current) return null
      const kratosError = error as KratosErrorWithResponse
      if (kratosError.response?.status === 401) {
        await clearPrivateSnapshots('session-expired', undefined, requestOwner)
        return null
      }
      console.error('Failed to check session:', error)
      // An unreachable Kratos server cannot prove that the previous identity signed out.
      if (!kratosError.response?.status || kratosError.response.status >= 500) await restoreOfflineIdentity()
      if (generation !== checkGeneration.current) return null
      setSession(null)
      return null
    } finally {
      if (sequence === checkSequence.current) setIsLoading(false)
    }
  }, [])

  React.useEffect(() => {
    const removeVerifier = setSessionVerifier(async () => !!(await checkSession())?.active)
    const invalidate = onPrivateCacheInvalidated((event) => {
      if (!(event.local && event.reason === 'account-change')) checkGeneration.current += 1
      setSession(null)
      setIsGuest(false)
      writeBrowserSetting('guestMode', null)
    })
    const reconnect = () => {
      void checkSession()
    }
    window.addEventListener('online', reconnect)
    return () => {
      removeVerifier()
      invalidate()
      window.removeEventListener('online', reconnect)
    }
  }, [checkSession])

  React.useEffect(() => {
    void checkSession()
  }, [checkSession])

  const login = React.useCallback(async (returnTo?: string) => {
    const generation = ++checkGeneration.current
    await resetLocalLogout()
    if (generation !== checkGeneration.current) return
    try {
      const { data: flow } = await kratos.createBrowserLoginFlow({
        returnTo: returnTo || new URL(import.meta.env.BASE_URL, window.location.origin).href
      })
      if (generation !== checkGeneration.current) return

      const microsoftNode = flow.ui.nodes.find(
        (node) =>
          node.group === 'oidc' &&
          node.attributes.node_type === 'input' &&
          (node.attributes as UiNodeInputAttributes).value === 'microsoft'
      )

      if (!microsoftNode) {
        console.error('Microsoft OIDC provider not found in flow')
        throw new Error('Microsoft OIDC provider is not configured in the local login flow')
      }

      const csrfNode = flow.ui.nodes.find((node) => (node.attributes as UiNodeInputAttributes).name === 'csrf_token')
      const csrfToken = csrfNode ? (csrfNode.attributes as UiNodeInputAttributes).value : ''

      const form = document.createElement('form')
      form.action = flow.ui.action
      form.method = 'POST'
      form.style.display = 'none'

      const providerInput = document.createElement('input')
      providerInput.name = 'provider'
      providerInput.value = 'microsoft'
      form.appendChild(providerInput)

      const csrfInput = document.createElement('input')
      csrfInput.name = 'csrf_token'
      csrfInput.value = csrfToken as string
      form.appendChild(csrfInput)

      document.body.appendChild(form)
      form.submit()
    } catch (error) {
      console.error('Failed to initiate login:', error)
      throw error
    }
  }, [])

  const logout = React.useCallback(async () => {
    const sequence = ++checkSequence.current
    setIsLoading(false)
    checkGeneration.current += 1
    setSession(null)
    setIsGuest(false)
    writeBrowserSetting('guestMode', null)
    const intentGeneration = currentAccountIntent()
    const intent = noteLocalLogout(session?.identity?.id ?? offline.identity?.identityId ?? null, intentGeneration)
    const intendedGeneration = checkGeneration.current
    if (!(await intent) || intendedGeneration !== checkGeneration.current) return
    const clearing = clearPrivateSnapshots('logout')
    await clearing
    if (sequence !== checkSequence.current || intentGeneration !== currentAccountIntent()) return
    const generation = checkGeneration.current
    if (!browserIsOnline()) return
    try {
      const { data } = await kratos.createBrowserLogoutFlow()
      if (generation !== checkGeneration.current) return
      window.location.href = data.logout_url
    } catch (error: unknown) {
      if (generation !== checkGeneration.current) return
      const kratosError = error as KratosErrorWithResponse
      if (kratosError.response?.status === 401) {
        setSession(null)
        window.location.href = `${import.meta.env.BASE_URL}auth/login`
        return
      }
      console.error('Logout failed:', error)
      setSession(null)
    }
  }, [session, offline.identity])

  const continueAsGuest = React.useCallback(() => {
    setIsGuest(true)
    writeBrowserSetting('guestMode', 'true')
  }, [])

  const exitGuestMode = React.useCallback(() => {
    setIsGuest(false)
    writeBrowserSetting('guestMode', null)
  }, [])

  if (isLoading) {
    return (
      <div className='flex min-h-screen items-center justify-center'>
        <div className='flex flex-col items-center gap-4'>
          <div className='h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent' />
          <p className='text-sm text-muted-foreground'>Loading...</p>
        </div>
      </div>
    )
  }

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        isLoading,
        isGuest,
        isOfflineReadonly,
        offlineIdentity: isOfflineReadonly ? offline.identity : null,
        session,
        login,
        logout,
        checkSession,
        continueAsGuest,
        exitGuestMode
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const context = React.useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
