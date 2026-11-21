import { useEffect, useState, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { createApi } from '@creata/contracts'
import type { Locale, Mode, Session } from './types'
import { tr } from './i18n'
import { AdminContext } from './admin-context'
import {adminDemo,demoRequest} from './demo'
const readLocale = (): Locale => { try { const l = localStorage.getItem('creata-admin-locale'); return l === 'kk' || l === 'en' ? l : 'ru' } catch { return 'ru' } }
export function AdminProvider({ mode, children }: { mode: Mode; children: ReactNode }) {
  const [locale, changeLocale] = useState<Locale>(readLocale)
  const [message, setMessage] = useState('')
  const queryClient = useQueryClient()
  const session = useQuery({ queryKey: ['session'], queryFn: () => (adminDemo?demoRequest:createApi())<Session>('/auth/me'), retry: 1 })
  const api = adminDemo?demoRequest:createApi('/api/creata/v1', () => session.data?.csrfToken)
  const setLocale = (next: Locale) => { changeLocale(next); try { localStorage.setItem('creata-admin-locale', next) } catch {} }
  useEffect(() => { document.documentElement.lang = locale }, [locale])
  useEffect(()=>{
    if(!adminDemo)return
    const changed=()=>{void queryClient.invalidateQueries()}
    const poll=window.setInterval(()=>{void queryClient.invalidateQueries({predicate:query=>['jobs','bank','articles','article','digests','deliveries','analytics','sources'].includes(String(query.queryKey[0]))})},1500)
    window.addEventListener('creata:pipeline-demo:change',changed);window.addEventListener('storage',changed)
    return()=>{clearInterval(poll);window.removeEventListener('creata:pipeline-demo:change',changed);window.removeEventListener('storage',changed)}
  },[queryClient])
  useEffect(() => { if (!message) return; const timer = window.setTimeout(() => setMessage(''), 5000); return () => clearTimeout(timer) }, [message])
  return <AdminContext.Provider value={{ mode, locale, setLocale, t: key => tr(key, locale), api, session: session.data, sessionLoading: session.isPending, sessionError: session.error, refreshSession: () => { void queryClient.invalidateQueries({ queryKey: ['session'] }) }, toast: setMessage }}>{children}{message && <div className='toast' role='status'>{message}<button onClick={() => setMessage('')} aria-label={tr('close', locale)}>×</button></div>}</AdminContext.Provider>
}
