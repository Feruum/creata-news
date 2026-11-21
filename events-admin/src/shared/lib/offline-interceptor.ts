import { Code, ConnectError, type Interceptor, type UnaryResponse } from '@connectrpc/connect'
import {
  browserIsOnline,
  clearPrivateSnapshots,
  getOfflineState,
  markOffline,
  OfflineReadUnavailable,
  offlineSnapshots,
  recordOfflineUpdate,
  requireVerifiedOnlineWrite
} from './offline-runtime'
import { ADMIN_OFFLINE_READ_METHODS } from './offline-snapshots'

export function isNetworkFailure(error: unknown): boolean {
  if (!browserIsOnline()) return true
  if (error instanceof TypeError) return true
  // Connect wraps a failed fetch in Unknown or Unavailable with the original cause.
  return error instanceof ConnectError && error.cause instanceof TypeError
}

export const offlineInterceptor: Interceptor = (next) => async (request) => {
  const method = `${request.service.typeName}/${request.method.name}`
  const approvedRead = ADMIN_OFFLINE_READ_METHODS.has(method)
  let lease = offlineSnapshots.lease()
  const isCurrentRequest = () => (lease ? offlineSnapshots.isCurrent(lease) : offlineSnapshots.lease() === null)

  const savedResponse = async (): Promise<UnaryResponse> => {
    if (request.stream) throw new OfflineReadUnavailable()
    const saved = await offlineSnapshots.read(method, request.message)
    if (!isCurrentRequest()) throw new ConnectError('The signed-in account changed.', Code.Canceled)
    if (!saved) throw new OfflineReadUnavailable()
    recordOfflineUpdate(saved.updatedAt)
    return {
      stream: false,
      service: request.service,
      method: request.method,
      header: new Headers(),
      trailer: new Headers(),
      message: saved.data as UnaryResponse['message']
    }
  }

  // Only the explicit read allowlist can use a snapshot. Unlisted calls require a live verified session.
  if (!approvedRead) lease = await requireVerifiedOnlineWrite()
  if (approvedRead && (!browserIsOnline() || getOfflineState().mode === 'offline-readonly')) return savedResponse()

  try {
    const response = await next(request)
    if (!isCurrentRequest()) throw new ConnectError('The signed-in account changed.', Code.Canceled)
    if (approvedRead && !request.stream && !response.stream && getOfflineState().mode === 'online') {
      const stored = await offlineSnapshots.save(lease, method, request.message, response.message)
      if (!isCurrentRequest()) throw new ConnectError('The signed-in account changed.', Code.Canceled)
      if (stored) recordOfflineUpdate(Date.now())
    }
    return response
  } catch (error) {
    if (!isCurrentRequest()) throw new ConnectError('The signed-in account changed.', Code.Canceled)
    if (error instanceof ConnectError && error.code === Code.Unauthenticated) {
      await clearPrivateSnapshots()
      throw error
    }
    if (isNetworkFailure(error)) {
      markOffline()
      if (approvedRead) return savedResponse()
    }
    throw error
  }
}
