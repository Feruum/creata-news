import type { Interceptor } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { offlineInterceptor } from './offline-interceptor'
import { onPrivateCacheInvalidated } from './offline-runtime'
import { ADMIN_OFFLINE_MAX_AGE } from './offline-snapshots'
import { mutationCategory } from './product-events'
import { getRequestErrorMessage } from './request-errors'
import { trackProductEvent } from './telemetry'

/**
 * Global error handling interceptor for Connect-RPC
 */
const errorInterceptor: Interceptor = (next) => async (req) => {
  return next(req)
}

/**
 * Connect-RPC transport configuration
 * - Uses binary format (Protobuf) in production for performance
 * - Falls back to JSON in development for easier debugging
 */
export const transport = createConnectTransport({
  baseUrl: import.meta.env.VITE_API_URL || 'http://localhost:5555',
  useBinaryFormat: import.meta.env.PROD,
  interceptors: [errorInterceptor, offlineInterceptor],
  fetch: (input, init) => {
    return fetch(input, {
      ...init,
      credentials: 'include',
      cache: 'no-store'
    })
  }
})

/**
 * Extract a user-friendly error message from Connect-RPC errors
 */
const getErrorMessage = getRequestErrorMessage

/**
 * TanStack Query client with global error handling and sensible defaults
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Data considered fresh for 5 minutes
      staleTime: 1000 * 60 * 5,
      // Matches the maximum lifetime of an offline snapshot.
      gcTime: ADMIN_OFFLINE_MAX_AGE,
      // Run the local snapshot query even when the browser reports offline.
      networkMode: 'always',
      // Retry failed requests once before showing error
      retry: 1,
      // Don't refetch on window focus in development
      refetchOnWindowFocus: import.meta.env.PROD
    },
    mutations: {
      // Offline actions must reject immediately and must never be queued/replayed.
      networkMode: 'always',
      retry: false
    }
  },
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Only show toast for queries that have already been cached
      // (to avoid showing errors on initial load which should be handled by components)
      if (query.state.data !== undefined && query.meta?.errorToast !== false) {
        toast.error(`Background sync failed: ${getErrorMessage(error)}`)
      }
    }
  }),
  mutationCache: new MutationCache({
    onMutate: (_variables, mutation) => {
      const category = mutationCategory(mutation.options.mutationKey)
      if (category) trackProductEvent('admin_action', { surface: 'admin', category, outcome: 'started' })
    },
    onSuccess: (_data, _variables, _context, mutation) => {
      const category = mutationCategory(mutation.options.mutationKey)
      if (category) trackProductEvent('admin_action', { surface: 'admin', category, outcome: 'success' })
    },
    onError: (error, _variables, _context, mutation) => {
      const category = mutationCategory(mutation.options.mutationKey)
      if (category) trackProductEvent('admin_action', { surface: 'admin', category, outcome: 'error' })
      // Forms that render errors inline can keep their controls unobstructed.
      if (mutation.meta?.errorToast !== false) toast.error(getErrorMessage(error))
    }
  })
})

onPrivateCacheInvalidated(() => {
  void queryClient.cancelQueries()
  queryClient.clear()
})

/**
 * Helper to create invalidation query keys for Connect-RPC services
 * Usage: invalidateQueries({ queryKey: createQueryKey(listEvents) })
 */
export function createServiceQueryKey(serviceMethod: { service: { typeName: string }; name: string }) {
  return [serviceMethod.service.typeName, serviceMethod.name] as const
}
