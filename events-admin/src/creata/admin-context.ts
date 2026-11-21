import { createContext } from 'react'
import { createApi } from '@creata/contracts'
import type { Locale, Mode, Session } from './types'
import type { CopyKey } from './i18n'
export interface AdminContextValue { locale: Locale; setLocale: (locale: Locale) => void; mode: Mode; t: (key: CopyKey) => string; api: ReturnType<typeof createApi>; session: Session | undefined; sessionLoading: boolean; sessionError: Error | null; refreshSession: () => void; toast: (message: string) => void }
export const AdminContext = createContext<AdminContextValue | null>(null)
