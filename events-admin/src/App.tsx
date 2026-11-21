import { TransportProvider } from '@connectrpc/connect-query'
import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { ProductAnalytics } from '@/shared/components/product-analytics'
import { PwaProvider } from '@/shared/components/pwa-controls'
import { queryClient, transport } from '@/shared/lib/api'
import { AuthProvider, useAuth } from '@/shared/lib/auth'
import { router } from '@/shared/lib/router'
import { Toaster } from '@/shared/ui/components/sonner'
import { ThemeProvider } from '@/shared/ui/components/theme-provider'

function InnerApp() {
  const auth = useAuth()
  const authorizationState = `${auth.isAuthenticated}:${auth.isOfflineReadonly}:${auth.isGuest}`
  const previousAuthorization = useRef<string | null>(null)
  useEffect(() => {
    if (previousAuthorization.current === authorizationState) return
    previousAuthorization.current = authorizationState
    void router.invalidate()
  }, [authorizationState])

  return (
    <>
      <ProductAnalytics />
      <RouterProvider router={router} context={{ auth }} />
    </>
  )
}

export function App() {
  return (
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider defaultTheme='dark' storageKey='vite-ui-theme'>
          <PwaProvider>
            <AuthProvider>
              <InnerApp />
              <Toaster richColors closeButton position='bottom-right' />
            </AuthProvider>
          </PwaProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </TransportProvider>
  )
}
