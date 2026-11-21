import { useInfiniteQuery, useQuery } from '@tanstack/react-query'
import type { SubmissionFilters } from '../types'
import { getSubmission, listSubmissions } from './client'
import { scheduleKeys } from './keys'

export function useScheduleSubmissions(filters: SubmissionFilters) {
  return useInfiniteQuery({
    queryKey: scheduleKeys.list(filters),
    queryFn: ({ pageParam, signal }) => listSubmissions(filters, pageParam, signal),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
    staleTime: 0,
    retry: false
  })
}

export function useScheduleSubmission(id: string) {
  return useQuery({
    queryKey: scheduleKeys.detail(id),
    queryFn: ({ signal }) => getSubmission(id, signal),
    staleTime: 0,
    retry: false
  })
}
