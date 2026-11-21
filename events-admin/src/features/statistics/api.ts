import { useQuery } from '@connectrpc/connect-query'
import {
  getEventActivityByYear,
  getEventCategoriesDistributionByMonth,
  getOrganizationTrends,
  getSundayOpenAnalytics,
  getSundayOpenAudienceStatistics,
  getTopPerformingClubs,
  getUserEngagementLevels
} from '@studyverse/proto'
import { attendanceAnalyticsQueryOptions } from '@/features/events'

export function useTopPerformingClubs(input: { limit: number; days: number }) {
  return useQuery(getTopPerformingClubs, input, attendanceAnalyticsQueryOptions)
}

export function useOrganizationTrends(input: { organizationIds: number[]; days: number }) {
  return useQuery(getOrganizationTrends, input, { enabled: input.organizationIds.length > 0 })
}

export function useEventCategoriesDistributionByMonth(input: { year: number; month: number }) {
  return useQuery(getEventCategoriesDistributionByMonth, input)
}

export function useUserEngagementLevels() {
  return useQuery(getUserEngagementLevels, {}, attendanceAnalyticsQueryOptions)
}

export function useEventActivityByYear(input: { year: number }) {
  return useQuery(getEventActivityByYear, input)
}

export function useSundayOpenAudienceStatistics() {
  return useQuery(getSundayOpenAudienceStatistics, {}, { retry: false })
}

export function useSundayOpenAnalytics() {
  return useQuery(getSundayOpenAnalytics, {}, { ...attendanceAnalyticsQueryOptions, retry: false })
}
