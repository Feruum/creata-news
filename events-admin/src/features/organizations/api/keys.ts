import { createConnectQueryKey } from '@connectrpc/connect-query'
import {
  getOrganization,
  getOrganizationMembershipHistory,
  listOrganizationMembers,
  listOrganizations,
  listOrganizationTypes,
  listRoleTemplatesByOrg,
  listRoleTemplatesByOrgType
} from '@studyverse/proto'

export function organizationMembershipHistoryKey(organizationId?: number) {
  return createConnectQueryKey({
    schema: getOrganizationMembershipHistory,
    ...(organizationId === undefined ? {} : { input: { organizationId } }),
    cardinality: 'finite'
  })
}

export function organizationsListKey() {
  return createConnectQueryKey({ schema: listOrganizations, cardinality: 'finite' })
}

export function organizationTypesListKey() {
  return createConnectQueryKey({ schema: listOrganizationTypes, cardinality: 'finite' })
}

export function organizationDetailKey(organizationId: number) {
  return createConnectQueryKey({ schema: getOrganization, input: { id: organizationId }, cardinality: 'finite' })
}

export function organizationMembersKey(organizationId: number) {
  return createConnectQueryKey({
    schema: listOrganizationMembers,
    input: { organizationId },
    cardinality: 'finite'
  })
}

export function roleTemplatesByOrgTypeKey(organizationTypeId: number) {
  return createConnectQueryKey({
    schema: listRoleTemplatesByOrgType,
    input: { organizationTypeId },
    cardinality: 'finite'
  })
}

export function organizationRoleTemplatesKey() {
  return createConnectQueryKey({ schema: listRoleTemplatesByOrg, cardinality: 'finite' })
}
