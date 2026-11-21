import { useQuery, useSuspenseQuery } from '@connectrpc/connect-query'
import { listActivityLog, listEntityActivityLog } from '@studyverse/proto'

export function useActivityLog(input: {
  organizationIds?: number[]
  actionFilters?: string[]
  after?: string
  before?: string
  actorId?: number
  page: number
  limit: number
}) {
  return useSuspenseQuery(listActivityLog, input)
}

export function useEntityActivityLog(input: {
  entityType: string
  entityId: number
  page: number
  limit: number
}) {
  return useSuspenseQuery(listEntityActivityLog, input)
}

export function useActivityLogLatest(input: {
  organizationIds?: number[]
  limit: number
}) {
  return useQuery(listActivityLog, { ...input, page: 1 })
}
