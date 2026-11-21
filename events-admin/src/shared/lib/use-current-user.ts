import { useQuery } from '@connectrpc/connect-query'
import { getMe, type User, type UserPermissions } from '@studyverse/proto'
import { useAuth } from './auth'

export interface CurrentUser {
  user: User | null
  permissions: UserPermissions | null
  isAdmin: boolean
  isStaff: boolean
  managedClubIds: number[]
  isLoading: boolean
  error: Error | null
  refetch: () => void
}

export function useCurrentUser(): CurrentUser {
  const { isAuthenticated, isOfflineReadonly, isLoading: isAuthLoading } = useAuth()

  const {
    data,
    isLoading: isQueryLoading,
    error,
    refetch
  } = useQuery(getMe, {}, { enabled: (isAuthenticated || isOfflineReadonly) && !isAuthLoading })

  const user = data?.user ?? null
  const permissions = data?.permissions ?? null

  return {
    user,
    permissions,
    isAdmin: permissions?.isAdmin ?? false,
    isStaff: permissions?.isStaff ?? false,
    managedClubIds: permissions?.managedClubIds ?? [],
    isLoading: isAuthLoading || isQueryLoading,
    error: error as Error | null,
    refetch
  }
}

export function useRequireAdmin(): CurrentUser & { hasAccess: boolean } {
  const currentUser = useCurrentUser()
  return {
    ...currentUser,
    hasAccess: currentUser.isAdmin
  }
}

export function useRequireStaff(): CurrentUser & { hasAccess: boolean } {
  const currentUser = useCurrentUser()
  return {
    ...currentUser,
    hasAccess: currentUser.isAdmin || currentUser.isStaff
  }
}

export function useCanManageClub(clubId: number): boolean {
  const { isAdmin, isStaff, managedClubIds } = useCurrentUser()
  if (isAdmin || isStaff) return true
  return managedClubIds.includes(clubId)
}
