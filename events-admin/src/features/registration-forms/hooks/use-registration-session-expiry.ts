import { useEffect, useState } from 'react'

export function useRegistrationSessionExpiry(expiresAt: string | undefined, checkSession: () => Promise<unknown>) {
  const [expiredAt, setExpiredAt] = useState<string>()
  useEffect(() => {
    const expires = Date.parse(expiresAt ?? '')
    if (!Number.isFinite(expires) || expires <= Date.now()) return
    let timer: number
    const checkExpiry = () => {
      const remaining = expires - Date.now()
      if (remaining <= 0) {
        setExpiredAt(expiresAt)
        void checkSession()
      } else timer = window.setTimeout(checkExpiry, Math.min(remaining, 2_147_483_647))
    }
    checkExpiry()
    return () => window.clearTimeout(timer)
  }, [expiresAt, checkSession])

  return !!expiresAt && (expiredAt === expiresAt || Date.parse(expiresAt) <= Date.now())
}
