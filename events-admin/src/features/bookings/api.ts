import { createConnectQueryKey, useMutation, useQuery, useSuspenseQuery } from '@connectrpc/connect-query'
import {
  approveBookingRequest,
  getAlternativeRooms,
  getBookingConflicts,
  getOrganization,
  getOrganizationType,
  getUser,
  listBookingRequests,
  rejectBookingRequest
} from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'

export function bookingRequestsListKey() {
  return createConnectQueryKey({ schema: listBookingRequests, cardinality: 'finite' })
}

export function bookingConflictsKey() {
  return createConnectQueryKey({ schema: getBookingConflicts, cardinality: 'finite' })
}

export function useBookingRequests(input: { status?: number; limit?: number; page?: number }) {
  return useSuspenseQuery(listBookingRequests, input)
}

export function useBookingConflicts(input: { fromDate: string; toDate: string }) {
  return useQuery(getBookingConflicts, input)
}

export function useAlternativeRooms(input: {
  startTime: string
  endTime: string
  minCapacity?: number
  preferredFloor?: number
  enabled: boolean
}) {
  return useQuery(
    getAlternativeRooms,
    {
      startTime: input.startTime,
      endTime: input.endTime,
      minCapacity: input.minCapacity,
      preferredFloor: input.preferredFloor
    },
    {
      enabled: input.enabled
    }
  )
}

export function useOrganizationDetails(input: { id: number; enabled?: boolean }) {
  return useQuery(getOrganization, { id: input.id }, { enabled: input.enabled })
}

export function useOrganizationTypeDetails(input: { id: number; enabled?: boolean }) {
  return useQuery(getOrganizationType, { id: input.id }, { enabled: input.enabled })
}

export function useUserDetails(input: { id: number }) {
  return useQuery(getUser, { id: input.id })
}

export function useApproveBookingRequest() {
  const queryClient = useQueryClient()
  return useMutation(approveBookingRequest, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bookingRequestsListKey() })
      queryClient.invalidateQueries({ queryKey: bookingConflictsKey() })
    }
  })
}

export function useRejectBookingRequest() {
  const queryClient = useQueryClient()
  return useMutation(rejectBookingRequest, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: bookingRequestsListKey() })
      queryClient.invalidateQueries({ queryKey: bookingConflictsKey() })
    }
  })
}
