'use client'
import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { createApi } from '@creata/contracts'
import type { AuthState } from '@/lib/models'
import {demoMode} from '@/demo/config'
type AuthContextValue = { auth: AuthState; loading: boolean; error: boolean; demo:boolean; refresh: () => Promise<void>; request: ReturnType<typeof createApi> }
const emptyAuth: AuthState = {user: null, providers: {google: false, telegram: false}}
const AuthContext = createContext<AuthContextValue | null>(null)
export function AuthProvider({children}: {children: ReactNode}) {
  const [auth,setAuth] = useState<AuthState>(emptyAuth)
  const [loading,setLoading] = useState(true)
  const [error,setError] = useState(false)
  const csrf = useRef<string | undefined>(undefined)
  const liveRequest = useCallback(createApi('/api/creata/v1', () => csrf.current), [])
  const request:ReturnType<typeof createApi>=useCallback(async<T,>(path:string,init:RequestInit={})=>{
    if(demoMode){const{createDemoRequest}=await import('@/demo/browser');return createDemoRequest()<T>(path,init)}
    return liveRequest<T>(path,init)
  },[liveRequest])
  const refresh = useCallback(async () => {
    setError(false)
    try { const result = await request<AuthState>('/auth/me'); csrf.current = result.csrfToken; setAuth(result) }
    catch { setError(true) }
    finally { setLoading(false) }
  }, [request])
  useEffect(() => { void refresh() }, [refresh])
  return <AuthContext.Provider value={{auth,loading,error,demo:demoMode,refresh,request}}>{children}</AuthContext.Provider>
}
export function useAuth() { const value = useContext(AuthContext); if (!value) throw new Error('AuthProvider required'); return value }
