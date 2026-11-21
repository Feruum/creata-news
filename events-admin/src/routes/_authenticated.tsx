import { createFileRoute, Outlet, redirect, useLocation } from '@tanstack/react-router'

import { AppSidebar } from '@/shared/components/app-sidebar'
import { SiteHeader } from '@/shared/components/site-header'
import { OfflineStatus } from '@/shared/components/offline-status'
import { OfflineViewError } from '@/shared/components/offline-view-error'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { SidebarInset, SidebarProvider } from '@/shared/ui/components/sidebar'
export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ context, location }) => {
    if (context.auth.isAuthenticated || context.auth.isOfflineReadonly) {
      return
    }

    if (context.auth.isGuest) {
      if (location.pathname === '/calendar') return
      throw redirect({ to: '/calendar' })
    }

    // Not authenticated and not in guest mode
    throw redirect({
      to: '/auth/login',
      search: {
        redirect: location.href
      }
    })
  },
  component: AuthenticatedLayout
})

function AuthenticatedLayout() {
  const location = useLocation()
  // Sync user from Kratos to app DB on first authenticated load
  useCurrentUser()

  return (
    <SidebarProvider
      className='h-svh overflow-hidden'
      style={
        {
          '--sidebar-width': 'calc(var(--spacing) * 72)',
          '--header-height': 'calc(var(--spacing) * 14)'
        } as React.CSSProperties
      }
    >
      <AppSidebar variant='inset' />
      <SidebarInset>
        <div className='@container/main flex min-h-0 flex-1 flex-col overflow-auto'>
          <SiteHeader />
          <OfflineStatus />
          <div className='mx-auto flex w-full max-w-7xl flex-1 flex-col px-4 py-6 sm:px-6 lg:px-8'>
            <OfflineViewError location={location.href}>
              <Outlet />
            </OfflineViewError>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
