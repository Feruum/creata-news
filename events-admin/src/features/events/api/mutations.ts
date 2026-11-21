import { useMutation } from '@connectrpc/connect-query'
import { cancelEvent, createEvent, deleteEvent, publishEvent, reviewEventBadges, updateEvent } from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'
import { eventDetailKey, eventPublishReadinessKey, eventsListKey } from './keys'

export function useCreateEvent() {
  const queryClient = useQueryClient()
  return useMutation(createEvent, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: eventsListKey() })
    }
  })
}

export function useUpdateEvent(errorToast = true) {
  const queryClient = useQueryClient()
  return useMutation(updateEvent, {
    meta: { errorToast },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: eventsListKey() })
      if (variables.id) {
        queryClient.invalidateQueries({ queryKey: eventDetailKey(variables.id) })
        queryClient.invalidateQueries({ queryKey: eventPublishReadinessKey({ eventId: variables.id }) })
      }
    }
  })
}

export function useReviewEventBadges() {
  const queryClient = useQueryClient()
  return useMutation(reviewEventBadges, {
    onSuccess: async (data, variables) => {
      const eventId = data.event?.id ?? variables.eventId
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: eventsListKey() }),
        ...(eventId !== undefined ? [queryClient.invalidateQueries({ queryKey: eventDetailKey(eventId) })] : [])
      ])
    }
  })
}

export function useDeleteEvent() {
  const queryClient = useQueryClient()
  return useMutation(deleteEvent, {
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: eventsListKey() })
      if (variables.id) {
        queryClient.invalidateQueries({ queryKey: eventDetailKey(variables.id) })
      }
    }
  })
}

export function usePublishEvent(errorToast = true) {
  const queryClient = useQueryClient()
  return useMutation(publishEvent, {
    meta: { errorToast },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: eventsListKey() })
      if (variables.eventId) {
        queryClient.invalidateQueries({ queryKey: eventDetailKey(variables.eventId) })
        queryClient.invalidateQueries({ queryKey: eventPublishReadinessKey({ eventId: variables.eventId }) })
      }
    }
  })
}

export function useCancelEvent() {
  const queryClient = useQueryClient()
  return useMutation(cancelEvent, {
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: eventsListKey() })
      if (variables.eventId) {
        queryClient.invalidateQueries({ queryKey: eventDetailKey(variables.eventId) })
      }
    }
  })
}
