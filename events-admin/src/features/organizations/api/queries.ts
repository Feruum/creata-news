import { createClient } from '@connectrpc/connect'
import { useQuery, useSuspenseQuery, useTransport } from '@connectrpc/connect-query'
import {
  getOrganization,
  getOrganizationMembershipHistory,
  getOrganizationSubscriberCount,
  getUserMembershipTimeline,
  listOrganizationMembers,
  listOrganizations,
  listOrganizationTypes,
  listRoleTemplatesByOrg,
  listRoleTemplatesByOrgType,
  listUsers,
  OrganizationMembersService,
  UsersService
} from '@studyverse/proto'
import { useQuery as useAsyncQuery } from '@tanstack/react-query'

export function useOrganizations(input: { limit: number; page: number }) {
  return useSuspenseQuery(listOrganizations, input)
}

export function useOrganizationTypes(input: { page?: number; limit?: number }) {
  return useSuspenseQuery(listOrganizationTypes, input)
}

export function useOrganization(organizationId: number) {
  return useSuspenseQuery(getOrganization, { id: organizationId })
}

export function useOrganizationPreview(organizationId: number, enabled = true) {
  return useQuery(getOrganization, { id: organizationId }, { enabled, retry: false })
}

export function useOrganizationSubscribers(organizationId: number, enabled = true) {
  return useQuery(getOrganizationSubscriberCount, { organizationId }, { enabled, retry: false })
}

export function useOrganizationTypesQuery(enabled = true) {
  return useQuery(listOrganizationTypes, { page: 1, limit: 1000 }, { enabled, retry: false })
}

export function useOrganizationRolesQuery(organizationId: number, organizationTypeId: number, enabled = true) {
  return useQuery(
    listRoleTemplatesByOrg,
    { organizationId, organizationTypeId },
    {
      enabled: enabled && organizationTypeId > 0,
      retry: false
    }
  )
}

export function useOrganizationMembers(organizationId: number) {
  return useSuspenseQuery(listOrganizationMembers, { organizationId })
}

export function useRoleTemplatesByOrgType(input: { organizationTypeId: number }) {
  return useSuspenseQuery(listRoleTemplatesByOrgType, input)
}

export function useRoleTemplatesByOrgTypeQuery(input: { organizationTypeId: number }) {
  return useQuery(listRoleTemplatesByOrgType, input)
}

export function useOrganizationMembersQuery(input: { organizationId: number }) {
  return useQuery(listOrganizationMembers, input)
}

export function useUsersList(input: { limit: number; enabled?: boolean }) {
  return useQuery(listUsers, { limit: input.limit }, { enabled: input.enabled })
}

export function useMemberLookup(value: string, enabled = true) {
  const transport = useTransport()
  const lookup = value.trim().replace(/^@/, '')
  return useAsyncQuery({
    queryKey: ['organization-member-lookup', lookup],
    enabled: enabled && !!lookup,
    retry: false,
    queryFn: async () => {
      const client = createClient(UsersService, transport)
      if (/^#\d+$/.test(lookup)) return client.getUser({ id: Number(lookup.slice(1)) })
      return client.getUserByUsername({ username: lookup })
    }
  })
}

export function useOrganizationMembersClient() {
  const transport = useTransport()
  return createClient(OrganizationMembersService, transport)
}

export function useOrganizationMembershipHistory(input: {
  organizationId: number
  actionFilter?: string
  limit?: number
  offset?: number
}) {
  return useQuery(getOrganizationMembershipHistory, input, { retry: false })
}

export function useUserMembershipTimeline(input: { userId: number; includeEnded: boolean }) {
  return useQuery(getUserMembershipTimeline, input)
}

export function useRoleTemplatesByOrg(input: { organizationId: number; organizationTypeId: number }) {
  return useSuspenseQuery(listRoleTemplatesByOrg, input)
}
