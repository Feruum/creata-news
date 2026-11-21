import { Code, ConnectError } from '@connectrpc/connect'
import { OfflineWriteBlocked } from './offline-runtime'

export type RequestErrorContext = 'save' | 'upload' | 'booking' | 'publish'

export function getRequestErrorMessage(error: unknown, context: RequestErrorContext = 'save'): string {
  if (error instanceof OfflineWriteBlocked) return error.message
  if (error instanceof ConnectError && error.cause instanceof OfflineWriteBlocked) return error.cause.message
  if (error instanceof ConnectError) {
    switch (error.code) {
      case Code.Unauthenticated:
        return 'Your session has expired. Sign in again and retry.'
      case Code.PermissionDenied:
        return 'You do not have permission to perform this action.'
      case Code.Unavailable:
      case Code.DeadlineExceeded:
        return 'Could not connect to the server. Check your connection and try again.'
      case Code.AlreadyExists:
        return context === 'booking'
          ? 'A booking request already exists. Refresh the room approval status.'
          : 'This item already exists. Refresh and try again.'
      case Code.FailedPrecondition:
        return context === 'booking' || context === 'publish'
          ? 'Room approval is required for this room and time. Check the booking status and try again.'
          : 'This action is not available yet. Refresh and try again.'
      case Code.InvalidArgument:
        return 'Check the event details and try again.'
      case Code.NotFound:
        return context === 'booking'
          ? 'The selected room could not be found. Choose a room and try again.'
          : 'This item could not be found. Refresh and try again.'
      case Code.ResourceExhausted:
        return 'Too many requests. Wait a moment and try again.'
    }
  }
  if (error instanceof TypeError || (error instanceof Error && /network|fetch|offline/i.test(error.message))) {
    return 'Could not connect to the server. Check your connection and try again.'
  }
  return context === 'upload'
    ? 'The cover image could not be uploaded. Please try again.'
    : context === 'booking'
      ? 'The room booking could not be requested. Please try again.'
      : context === 'publish'
        ? 'The event could not be published. Refresh room approval and try again.'
        : 'Your changes could not be saved. Please try again.'
}
