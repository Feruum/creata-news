import { useIsMutating, useMutation, useQueryClient } from '@tanstack/react-query'
import type { Session, User } from './types'
import { useAdmin } from './hooks'

export type AdminTheme = NonNullable<User['adminTheme']>

export function normalizeAdminTheme(value: unknown): AdminTheme {
  return value === 'dark' || value === 'cats' ? value : 'light'
}

/** The session is the source of truth in both applications, never a browser-global preference. */
export function useAdminTheme() {
  const { api, session } = useAdmin()
  const client = useQueryClient()
  const userId = session?.user?.id
  const mutationKey = ['adminTheme', userId]
  const mutation = useMutation({
    mutationKey,
    mutationFn: (adminTheme: AdminTheme) => api<User>('/account', {
      method: 'PATCH', body: JSON.stringify({ adminTheme }),
    }),
    onSuccess: user => {
      // A delayed response from a previous account must not replace the active session.
      client.setQueryData<Session>(['session'], current => current?.user?.id === user.id
        ? { ...current, user }
        : current)
    },
  })
  const pending = useIsMutating({ mutationKey }) > 0
  return {
    theme: normalizeAdminTheme(session?.user?.adminTheme),
    setTheme: mutation.mutate,
    error: mutation.error,
    pending,
  }
}
