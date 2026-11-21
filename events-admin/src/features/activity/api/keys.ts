import { createConnectQueryKey } from '@connectrpc/connect-query'
import { listActivityLog, listEntityActivityLog } from '@studyverse/proto'

export function activityLogListKey(input?: {
  organizationIds?: number[]
  actionFilters?: string[]
  after?: string
  before?: string
  actorId?: number
  page?: number
  limit?: number
}) {
  if (input) {
    return createConnectQueryKey({ schema: listActivityLog, input, cardinality: 'finite' })
  }
  return createConnectQueryKey({ schema: listActivityLog, cardinality: 'finite' })
}

export function entityActivityLogKey(input: {
  entityType: string
  entityId: number
  page?: number
  limit?: number
}) {
  return createConnectQueryKey({ schema: listEntityActivityLog, input, cardinality: 'finite' })
}
