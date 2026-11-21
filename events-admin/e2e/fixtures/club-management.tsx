import { create } from '@bufbuild/protobuf'
import { Code, ConnectError, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  Capability,
  OrganizationMemberSchema,
  OrganizationMembersService,
  OrganizationSchema,
  OrganizationSubscriptionsService,
  OrganizationsService,
  OrganizationTypesService,
  type RoleTemplate,
  RoleTemplateSchema,
  RoleTemplatesService,
  UserSchema,
  UsersService
} from '@studyverse/proto'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { NuqsTestingAdapter } from 'nuqs/adapters/testing'
import { Suspense, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import { columns, OrganizationInfoModalControlled, useOrganizations } from '@/features/organizations'
import { DataTable } from '@/shared/components/admin-data-table'

const params = new URLSearchParams(window.location.search)
const user = create(UserSchema, { id: 142, username: '250875', firstName: 'Madi', lastName: 'Talgat' })
let organization = create(OrganizationSchema, {
  id: 20,
  title: 'Campus Club',
  organizationTypeId: 1,
  status: 1,
  instagram: 'campusclub',
  imageUrl: '/club-avatar.svg',
  description: '[{"type":"p","children":[{"text":"Campus technology community"}]}]',
  createdAt: '2026-10-01T12:00:00Z',
  updatedAt: '2026-10-02T12:00:00Z',
  website: 'https://campusclub.example.com',
  telegramChannel: 'campusclub',
  slug: 'campusclub'
})
const otherOrganization = create(OrganizationSchema, {
  id: 21,
  title: 'Design Club',
  organizationTypeId: 1,
  status: 1,
  description: '[{"type":"p","children":[{"text":"A community for campus designers"}]}]',
  createdAt: '2026-09-12T12:00:00Z'
})
let updateAttempts = 0
let typeRequests = 0
let memberExists = !params.has('empty')
const extraRoles: RoleTemplate[] = []
const removedRoleIds = new Set<number>()
let sharedRole = create(RoleTemplateSchema, {
  id: 8,
  organizationTypeId: 1,
  key: 'president',
  displayName: 'President',
  isActive: true,
  maxHolders: 3,
  capabilities: [Capability.MANAGE_SETTINGS]
})
let role = create(RoleTemplateSchema, {
  id: 7,
  organizationTypeId: 1,
  organizationId: 20,
  key: 'coordinator',
  displayName: 'Coordinator',
  isActive: true,
  capabilities: [Capability.CREATE_EVENT],
  parentTemplateId: 8,
  maxHolders: 5
})
let member = create(OrganizationMemberSchema, {
  id: 15,
  organizationId: 20,
  userId: 142,
  roleTemplateId: 7,
  firstName: 'Madi',
  lastName: 'Talgat',
  username: '250875',
  validUntil: '2035-01-01T00:00:00Z'
})
const requests: { method: string; input: unknown }[] = []
function record(method: string, input: unknown) {
  requests.push({ method, input })
  document.querySelector('[data-testid=requests]')!.textContent = JSON.stringify(requests)
}
const transport = createRouterTransport(({ service }) => {
  service(OrganizationsService, {
    listOrganizations: () => ({ organizations: [organization, otherOrganization], total: 2 }),
    getOrganization: ({ id }) => {
      if (params.has('loadError')) throw new ConnectError('Could not load club', Code.Unavailable)
      return {
        organization: id === 21 ? otherOrganization : organization,
        canEdit: !params.has('readonly') && !params.has('membersOnly'),
        canManageMembers: !params.has('readonly') && !params.has('settingsOnly')
      }
    },
    updateOrganization: async (input) => {
      record('UpdateOrganization', input)
      if (params.has('slowSave')) await new Promise((resolve) => setTimeout(resolve, 1800))
      if (params.get('saveError') === 'once' && updateAttempts++ === 0)
        throw new ConnectError('Could not save club', Code.Unavailable)
      organization = create(OrganizationSchema, {
        ...organization,
        ...input,
        title: input.title ?? organization.title,
        $typeName: organization.$typeName
      })
      return { organization }
    },
    uploadOrganizationImage: (input) => {
      record('UploadOrganizationImage', { organizationId: input.organizationId, contentType: input.contentType })
      if (params.has('uploadError')) throw new ConnectError('Could not upload image', Code.Unavailable)
      return { publicUrl: '/club-avatar.svg', objectKey: 'club-avatar.svg' }
    },
    deleteOrganization: (input) => {
      record('DeleteOrganization', input)
      return { success: true }
    },
    getOrganizationMembershipHistory: () => {
      if (params.has('historyError')) throw new ConnectError('History unavailable', Code.Unavailable)
      return { entries: [], total: 0 }
    }
  })
  service(OrganizationSubscriptionsService, {
    getOrganizationSubscriberCount: (input) => {
      record('GetOrganizationSubscriberCount', input)
      return { count: 48 }
    }
  })
  service(OrganizationTypesService, {
    listOrganizationTypes: () => {
      record('ListOrganizationTypes', {})
      if (params.has('typesError') || (params.has('typesAfterError') && typeRequests++ > 0))
        throw new ConnectError('Types unavailable', Code.Unavailable)
      return {
        organizationTypes: [
          { id: 1, title: 'Club' },
          { id: 2, title: 'Community' }
        ],
        total: 2
      }
    }
  })
  service(RoleTemplatesService, {
    listRoleTemplatesByOrg: () => ({
      roleTemplates: [sharedRole, role, ...extraRoles].filter((entry) => !removedRoleIds.has(entry.id))
    }),
    listRoleTemplatesByOrgType: () => ({ roleTemplates: [sharedRole, role] }),
    createRoleTemplate: (input) => {
      record('CreateRoleTemplate', input)
      const createdRole = create(RoleTemplateSchema, { ...input, id: 9 + extraRoles.length })
      extraRoles.push(createdRole)
      return { roleTemplate: createdRole }
    },
    deleteRoleTemplate: (input) => {
      record('DeleteRoleTemplate', input)
      removedRoleIds.add(input.id)
      return { success: true }
    },
    updateRoleTemplate: (input) => {
      record('UpdateRoleTemplate', input)
      const current = input.id === sharedRole.id ? sharedRole : role
      const updated = create(RoleTemplateSchema, {
        ...current,
        ...input,
        parentTemplateId: input.clearParentTemplate ? undefined : (input.parentTemplateId ?? current.parentTemplateId),
        maxHolders: input.clearMaxHolders ? undefined : (input.maxHolders ?? current.maxHolders),
        $typeName: current.$typeName
      })
      if (input.id === sharedRole.id) sharedRole = updated
      else role = updated
      return { roleTemplate: updated }
    },
    setTemplateCapabilities: (input) => {
      record('SetTemplateCapabilities', input)
      if (input.roleTemplateId === sharedRole.id)
        sharedRole = create(RoleTemplateSchema, { ...sharedRole, capabilities: input.capabilities })
      else role = create(RoleTemplateSchema, { ...role, capabilities: input.capabilities })
      return { roleTemplate: input.roleTemplateId === sharedRole.id ? sharedRole : role }
    }
  })
  service(OrganizationMembersService, {
    listOrganizationMembers: () => ({
      members: memberExists ? [member] : [],
      total: memberExists ? 1 : 0
    }),
    addOrganizationMember: (input) => {
      record('AddOrganizationMember', input)
      memberExists = true
      member = create(OrganizationMemberSchema, {
        ...member,
        roleTemplateId: input.roleTemplateId,
        validUntil: input.validUntil
      })
      return { member }
    },
    updateOrganizationMember: (input) => {
      record('UpdateOrganizationMember', input)
      member = create(OrganizationMemberSchema, { ...member, ...input, $typeName: member.$typeName })
      return { member }
    },
    removeOrganizationMember: (input) => {
      record('RemoveOrganizationMember', input)
      memberExists = false
      return { success: true }
    }
  })
  service(UsersService, {
    getMe: () => ({
      user,
      permissions: { isStaff: !params.has('readonly') && !params.has('membersOnly') && !params.has('settingsOnly') }
    }),
    getUserByUsername: (input) => {
      record('GetUserByUsername', input)
      return { user }
    },
    getUser: (input) => {
      record('GetUser', input)
      return { user }
    }
  })
})
const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
function Fixture() {
  const [selectedId, setSelectedId] = useState<number | null>(params.has('closed') ? null : 20)
  const { data } = useOrganizations({ page: 1, limit: 20 })
  return (
    <>
      <main className='p-6'>
        <h1 className='text-2xl font-semibold'>Club management · admin</h1>
        <p>Local verification data</p>
        <button type='button' onClick={() => setSelectedId(20)}>
          Open club
        </button>
        <div className='mt-8'>
          <DataTable
            data={data.organizations}
            columns={columns}
            totalCount={data.total}
            entityName='clubs'
            searchPlaceholder='Search clubs...'
            enableColumnResizing={params.has('resizeColumns')}
            isLoading={params.has('loadingTable')}
            onRowClick={(club) => setSelectedId(club.id)}
            activeRowId={selectedId === null ? null : String(selectedId)}
          />
        </div>
      </main>
      <OrganizationInfoModalControlled
        organizationId={selectedId}
        onOrganizationChange={setSelectedId}
        onClose={() => setSelectedId(null)}
      />
      <output data-testid='requests' className='sr-only'>
        []
      </output>
    </>
  )
}
createRoot(document.getElementById('root')!).render(
  <NuqsTestingAdapter hasMemory>
    <QueryClientProvider client={queryClient}>
      <TransportProvider transport={transport}>
        <Suspense fallback={<p>Loading club</p>}>
          <Fixture />
        </Suspense>
        <Toaster />
      </TransportProvider>
    </QueryClientProvider>
  </NuqsTestingAdapter>
)
