import { useContext } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AdminContext } from './admin-context'
import type { Category, List, User } from './types'
export function useAdmin() { const ctx = useContext(AdminContext); if (!ctx) throw new Error('Admin context is missing'); return ctx }
export function useCategories() { const { api } = useAdmin(); return useQuery({ queryKey: ['categories'], queryFn: () => api<List<Category>>('/public/categories'), staleTime: 60_000 }) }
export function useStaff() {
  const { api } = useAdmin()
  return useQuery({ queryKey: ['staff'], queryFn: () => api<List<Pick<User, 'id' | 'name' | 'roles' | 'verticals'>>>('/admin/staff') })
}
