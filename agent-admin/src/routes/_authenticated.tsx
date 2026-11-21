import { createFileRoute, Outlet } from '@tanstack/react-router'
import { AuthProvider, useAuth } from '@/shared/lib/auth'
import { Sidebar } from '@/shared/components/sidebar'
import { Header } from '@/shared/components/header'

function AuthenticatedLayout() {
  const { isAuthenticated, isAdmin, isLoading, login } = useAuth()

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!isAuthenticated) {
    login()
    return null
  }

  if (!isAdmin) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="text-center space-y-2">
          <p className="text-lg font-medium">Access Denied</p>
          <p className="text-sm text-muted-foreground">You don't have admin privileges.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated')({
  component: () => (
    <AuthProvider>
      <AuthenticatedLayout />
    </AuthProvider>
  ),
})
