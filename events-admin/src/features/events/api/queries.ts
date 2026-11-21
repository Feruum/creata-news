import { useQuery, useSuspenseQuery } from '@connectrpc/connect-query'
import {
  CheckInPhase,
  EventStatus,
  type GetEventPublishReadinessRequest,
  getCheckInInfo,
  getEvent,
  getEventAttendance,
  getEventPublishReadiness,
  getEventRegistrations,
  listCategories,
  listEventsForAdmin,
  listOrganizations,
  listRooms,
  sundayopenConnect
} from '@studyverse/proto'

const attendanceFreshnessOptions = {
  staleTime: 0,
  refetchOnMount: 'always' as const,
  refetchOnWindowFocus: 'always' as const,
  refetchIntervalInBackground: false
}

export const attendanceAnalyticsQueryOptions = {
  ...attendanceFreshnessOptions,
  refetchInterval: 15_000
}

export function useEventsForAdmin(input: { limit: number; page: number }) {
  return useSuspenseQuery(listEventsForAdmin, input, attendanceAnalyticsQueryOptions)
}

export function useEvent(eventId: number) {
  return useSuspenseQuery(
    getEvent,
    { id: eventId },
    {
      ...attendanceFreshnessOptions,
      refetchInterval: (query) => {
        const event = query.state.data?.event
        const phase = event?.checkInPhase
        // Discover activation when the card was opened before the check-in window.
        const awaitingCheckIn = event?.status === EventStatus.PUBLISHED && phase === CheckInPhase.NONE
        return awaitingCheckIn || phase === CheckInPhase.LIVE || phase === CheckInPhase.GRACE ? 5000 : false
      }
    }
  )
}

export function useEventPublishReadiness(
  eventId: number,
  enabled: boolean,
  proposal?: Pick<GetEventPublishReadinessRequest, 'proposedStartTime' | 'proposedEndTime' | 'proposedLocation'>
) {
  return useQuery(
    getEventPublishReadiness,
    { eventId, ...proposal },
    { enabled, staleTime: 0, meta: { errorToast: false } }
  )
}

export function useEventCategories(input: { page: number; limit: number }) {
  return useSuspenseQuery(listCategories, input)
}

export function useRooms(input: { bookableOnly: boolean; page: number; limit: number }) {
  return useQuery(listRooms, input)
}

export function useOrganizations(input: { page: number; limit: number }) {
  return useQuery(listOrganizations, input)
}

export function useEventRegistrations(
  input: { eventId: number; page?: number; limit?: number },
  isCheckInActive = false
) {
  return useSuspenseQuery(getEventRegistrations, input, {
    ...attendanceFreshnessOptions,
    refetchInterval: isCheckInActive ? 5000 : false
  })
}

export function useEventAttendance(input: { eventId: number }, isCheckInActive = false) {
  return useQuery(getEventAttendance, input, {
    ...attendanceFreshnessOptions,
    refetchInterval: isCheckInActive ? 5000 : false
  })
}

export function useExternalRegistrations(input: { eventId: number; page: number; limit: number }) {
  return useQuery(sundayopenConnect.getExternalRegistrations, input, attendanceFreshnessOptions)
}

export function useEventCheckInInfo(eventId: number, enabled: boolean, refetchInterval?: number) {
  return useQuery(
    getCheckInInfo,
    { eventId },
    {
      enabled,
      refetchInterval
    }
  )
}
