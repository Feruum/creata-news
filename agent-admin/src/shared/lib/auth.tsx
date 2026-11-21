import type { Session } from '@ory/client'
import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { kratos } from './kratos'

const ADMIN_EMAILS = [
  '242588@astanait.edu.kz',
]

interface AuthContextType {
  isAuthenticated: boolean
  isAdmin: boolean
  isLoading: boolean
  session: Session | null
  login: () => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    kratos.toSession()
      .then(({ data }) => setSession(data))
      .catch(() => setSession(null))
      .finally(() => setIsLoading(false))
  }, [])

  const login = useCallback(() => {
    window.location.href = `${import.meta.env.VITE_KRATOS_PUBLIC_URL || 'http://localhost:4433'}/self-service/login/browser`
  }, [])

  const logout = useCallback(async () => {
    try {
      const { data } = await kratos.createBrowserLogoutFlow()
      window.location.href = data.logout_url
    } catch {
      setSession(null)
    }
  }, [])

  const email = (session?.identity?.traits as Record<string, unknown>)?.email as string | undefined
  const isAdmin = !!email && ADMIN_EMAILS.includes(email)

  return (
    <AuthContext.Provider value={{
      isAuthenticated: !!session?.active,
      isAdmin,
      isLoading,
      session,
      login,
      logout,
    }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
