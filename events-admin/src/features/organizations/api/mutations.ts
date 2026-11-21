import { createClient } from '@connectrpc/connect'
import { createConnectQueryKey, useMutation as useConnectMutation, useTransport } from '@connectrpc/connect-query'
import {
  type Capability,
  createOrganization,
  deleteOrganization,
  getMe,
  getOrganization,
  getUserMembershipTimeline,
  listOrganizationMembers,
  OrganizationMembersService,
  OrganizationsService,
  OrganizationTypesService,
  RoleTemplatesService,
  updateOrganization
} from '@studyverse/proto'
import { type QueryClient, useMutation, useQueryClient } from '@tanstack/react-query'
import { getErrorMessage } from '@/shared/lib/form-utils'
import {
  organizationDetailKey,
  organizationMembershipHistoryKey,
  organizationMembersKey,
  organizationRoleTemplatesKey,
  organizationsListKey,
  organizationTypesListKey,
  roleTemplatesByOrgTypeKey
} from './keys'

function invalidateOrganizationAccess(queryClient: QueryClient, organizationId?: number) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: organizationsListKey() }),
    queryClient.invalidateQueries({
      queryKey:
        organizationId === undefined
          ? createConnectQueryKey({ schema: getOrganization, cardinality: 'finite' })
          : organizationDetailKey(organizationId)
    }),
    queryClient.invalidateQueries({
      queryKey:
        organizationId === undefined
          ? createConnectQueryKey({ schema: listOrganizationMembers, cardinality: 'finite' })
          : organizationMembersKey(organizationId)
    }),
    queryClient.invalidateQueries({ queryKey: organizationMembershipHistoryKey(organizationId) }),
    queryClient.invalidateQueries({ queryKey: createConnectQueryKey({ schema: getMe, cardinality: 'finite' }) }),
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({ schema: getUserMembershipTimeline, cardinality: 'finite' })
    })
  ])
}

export function useCreateOrganization() {
  const queryClient = useQueryClient()
  return useConnectMutation(createOrganization, {
    meta: { errorToast: false },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationsListKey() })
    }
  })
}

export function useUpdateOrganization() {
  const queryClient = useQueryClient()
  return useConnectMutation(updateOrganization, {
    meta: { errorToast: false },
    onSuccess: (_, input) =>
      Promise.all([
        invalidateOrganizationAccess(queryClient, input.id),
        queryClient.invalidateQueries({ queryKey: organizationRoleTemplatesKey() })
      ])
  })
}

export function useDeleteOrganization() {
  const queryClient = useQueryClient()
  return useConnectMutation(deleteOrganization, {
    onSuccess: (_, input) => invalidateOrganizationAccess(queryClient, input.id)
  })
}

export function useUpdateOrganizationMember(organizationId: number) {
  const queryClient = useQueryClient()
  const client = createClient(OrganizationMembersService, useTransport())
  return useMutation({
    mutationFn: (input: { id: number; roleTemplateId: number; validUntil: string }) =>
      client.updateOrganizationMember(input),
    onSuccess: () => invalidateOrganizationAccess(queryClient, organizationId)
  })
}

export function useRemoveOrganizationMember(organizationId: number) {
  const queryClient = useQueryClient()
  const client = createClient(OrganizationMembersService, useTransport())
  return useMutation({
    mutationFn: (id: number) => client.removeOrganizationMember({ id }),
    onSuccess: () => invalidateOrganizationAccess(queryClient, organizationId)
  })
}

export function useOrganizationTypesClient() {
  const transport = useTransport()
  return createClient(OrganizationTypesService, transport)
}

export function useOrganizationsClient() {
  const transport = useTransport()
  return createClient(OrganizationsService, transport)
}

export function useCreateOrganizationType() {
  const queryClient = useQueryClient()
  const transport = useTransport()
  const client = createClient(OrganizationTypesService, transport)
  return useMutation({
    mutationFn: (title: string) => client.createOrganizationType({ title }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationTypesListKey() })
    }
  })
}

export function useUpdateOrganizationType() {
  const queryClient = useQueryClient()
  const transport = useTransport()
  const client = createClient(OrganizationTypesService, transport)
  return useMutation({
    mutationFn: ({ id, title }: { id: number; title: string }) => client.updateOrganizationType({ id, title }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationTypesListKey() })
    }
  })
}

export function useDeleteOrganizationType() {
  const queryClient = useQueryClient()
  const transport = useTransport()
  const client = createClient(OrganizationTypesService, transport)
  return useMutation({
    mutationFn: (id: number) => client.deleteOrganizationType({ id }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationTypesListKey() })
    }
  })
}

export function useInvalidateRoleTemplates(organizationTypeId: number, organizationId?: number) {
  const queryClient = useQueryClient()
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: roleTemplatesByOrgTypeKey(organizationTypeId) }),
      queryClient.invalidateQueries({ queryKey: organizationRoleTemplatesKey() }),
      invalidateOrganizationAccess(queryClient, organizationId)
    ])
}

export function useCreateRoleTemplate(organizationTypeId: number, organizationId?: number) {
  const invalidate = useInvalidateRoleTemplates(organizationTypeId, organizationId)
  const transport = useTransport()
  const client = createClient(RoleTemplatesService, transport)
  return useMutation({
    mutationFn: (params: {
      organizationId?: number
      key: string
      displayName: string
      description?: string
      parentTemplateId?: number
      sortOrder: number
      maxHolders?: number
      isRequired: boolean
      isActive?: boolean
      capabilities: Capability[]
    }) =>
      client.createRoleTemplate({
        organizationTypeId,
        organizationId: params.organizationId ?? organizationId,
        key: params.key,
        displayName: params.displayName,
        description: params.description,
        parentTemplateId: params.parentTemplateId,
        sortOrder: params.sortOrder,
        maxHolders: params.maxHolders,
        isRequired: params.isRequired,
        isActive: params.isActive ?? true,
        capabilities: params.capabilities
      }),
    onSuccess: invalidate
  })
}

export function useUpdateRoleTemplate(organizationTypeId: number, organizationId?: number) {
  const invalidate = useInvalidateRoleTemplates(organizationTypeId, organizationId)
  const transport = useTransport()
  const client = createClient(RoleTemplatesService, transport)
  return useMutation({
    mutationFn: async (params: {
      id: number
      displayName: string
      description?: string
      parentTemplateId?: number
      clearParentTemplate?: boolean
      sortOrder?: number
      maxHolders?: number
      clearMaxHolders?: boolean
      isRequired: boolean
      isActive?: boolean
      capabilities: Capability[]
    }) => {
      await client.updateRoleTemplate({
        id: params.id,
        displayName: params.displayName,
        description: params.description,
        parentTemplateId: params.parentTemplateId,
        clearParentTemplate: params.clearParentTemplate,
        sortOrder: params.sortOrder,
        maxHolders: params.maxHolders,
        clearMaxHolders: params.clearMaxHolders,
        isRequired: params.isRequired,
        isActive: params.isActive
      })
      try {
        return await client.setTemplateCapabilities({ roleTemplateId: params.id, capabilities: params.capabilities })
      } catch (error) {
        throw new Error(`Role details were saved, but capabilities could not be saved: ${getErrorMessage(error)}`, {
          cause: error
        })
      }
    },
    onSettled: invalidate
  })
}

export function useDeleteRoleTemplate(organizationTypeId: number, organizationId?: number) {
  const invalidate = useInvalidateRoleTemplates(organizationTypeId, organizationId)
  const transport = useTransport()
  const client = createClient(RoleTemplatesService, transport)
  return useMutation({
    mutationFn: (id: number) => client.deleteRoleTemplate({ id }),
    onSuccess: invalidate
  })
}

export function useAddOrganizationMember(organizationId: number) {
  const queryClient = useQueryClient()
  const transport = useTransport()
  const client = createClient(OrganizationMembersService, transport)
  return useMutation({
    mutationFn: (params: { userId: number; roleTemplateId: number; validUntil?: string }) =>
      client.addOrganizationMember({
        organizationId,
        userId: params.userId,
        roleTemplateId: params.roleTemplateId,
        validUntil: params.validUntil
      }),
    onSuccess: () => invalidateOrganizationAccess(queryClient, organizationId)
  })
}
