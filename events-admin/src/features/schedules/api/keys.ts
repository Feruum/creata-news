import type { SubmissionFilters } from '../types'

export const scheduleKeys = {
  all: ['schedule-moderation'] as const,
  list: (filters: SubmissionFilters) => [...scheduleKeys.all, 'list', filters] as const,
  detail: (id: string) => [...scheduleKeys.all, 'detail', id] as const
}
