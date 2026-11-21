import { createClient } from '@connectrpc/connect'
import { createConnectQueryKey, useTransport } from '@connectrpc/connect-query'
import {
  AchievementsService,
  EventsService,
  OrganizationMembersService,
  SearchService,
  UsersService
} from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'
import { useMemo } from 'react'

export function useAchievementClient() {
  const transport = useTransport()
  return useMemo(() => createClient(AchievementsService, transport), [transport])
}

export function useRefreshAchievements() {
  const queryClient = useQueryClient()
  return async () => {
    await Promise.all(
      [AchievementsService, UsersService, OrganizationMembersService, EventsService, SearchService].map((schema) =>
        queryClient.invalidateQueries({ queryKey: createConnectQueryKey({ schema, cardinality: 'finite' }) })
      )
    )
  }
}
