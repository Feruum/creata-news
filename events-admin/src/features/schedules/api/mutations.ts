import { useMutation, useQueryClient } from '@tanstack/react-query'
import { reviewSubmission, ScheduleApiError } from './client'
import { scheduleKeys } from './keys'

export function useReviewSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: reviewSubmission,
    retry: false,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: scheduleKeys.all })
    },
    onError: async (error) => {
      if (error instanceof ScheduleApiError && error.status === 409) {
        await queryClient.invalidateQueries({ queryKey: scheduleKeys.all })
      }
    }
  })
}
