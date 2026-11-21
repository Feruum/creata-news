import type { Session } from '@ory/client'

export function getLoginReturnUrl(redirect: string | undefined, basePath: string, origin: string): string {
  const base = new URL(basePath, origin)
  if (!redirect) return base.href
  const target = new URL(redirect, base)
  if (target.origin !== base.origin) return base.href
  const prefix = base.pathname.replace(/\/$/, '')
  if (prefix && target.pathname !== prefix && !target.pathname.startsWith(`${prefix}/`)) {
    target.pathname = `${prefix}/${target.pathname.replace(/^\//, '')}`
  }
  return target.href
}

// Extract traits type from identity schema
export interface KratosIdentityTraits {
  email: string
  name?: {
    first?: string
    last?: string
  }
}

// Helper to get user display name from session
export function getUserDisplayName(session: Session | null): string {
  if (!session) return 'Guest'
  const traits = session.identity?.traits as KratosIdentityTraits | undefined
  if (!traits) return 'User'
  if (traits.name?.first && traits.name?.last) {
    return `${traits.name.first} ${traits.name.last}`
  }
  if (traits.name?.first) return traits.name.first
  return traits.email?.split('@')[0] || 'User'
}

// Helper to get user initials for avatar
export function getUserInitials(session: Session | null): string {
  if (!session) return '?'
  const traits = session.identity?.traits as KratosIdentityTraits | undefined
  if (!traits) return '?'
  if (traits.name?.first && traits.name?.last) {
    return `${traits.name.first[0]}${traits.name.last[0]}`.toUpperCase()
  }
  return traits.email?.substring(0, 2).toUpperCase() || '??'
}

// Helper to get user email
export function getUserEmail(session: Session | null): string {
  if (!session) return ''
  const traits = session.identity?.traits as KratosIdentityTraits | undefined
  return traits?.email || ''
}
