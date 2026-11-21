import type { Capability, RoleTemplate } from '@studyverse/proto'

interface RoleFormValues {
  id: number
  displayName: string
  description: string
  parentTemplateId: string
  sortOrder: string
  maxHolders: string
  isRequired: boolean
  isActive: boolean
  capabilities: Capability[]
}

function parseInteger(value: string, label: string, min: number) {
  const number = Number(value)
  if (!value.trim() || !Number.isInteger(number) || number < min || number > 2147483647) {
    throw new Error(`${label} must be a whole number ${min === 1 ? 'greater than zero' : 'within the allowed range'}`)
  }
  return number
}

export function getRoleUpdateInput(values: RoleFormValues) {
  return {
    id: values.id,
    displayName: values.displayName.trim(),
    description: values.description,
    parentTemplateId: values.parentTemplateId ? parseInteger(values.parentTemplateId, 'Parent role', 1) : undefined,
    clearParentTemplate: !values.parentTemplateId,
    sortOrder: parseInteger(values.sortOrder, 'Sort order', -2147483648),
    maxHolders: values.maxHolders ? parseInteger(values.maxHolders, 'Max members', 1) : undefined,
    clearMaxHolders: !values.maxHolders,
    isRequired: values.isRequired,
    isActive: values.isActive,
    capabilities: values.capabilities
  }
}

export function canEditRoleTemplate(
  role: Pick<RoleTemplate, 'organizationId'>,
  permissions: { organizationId?: number; canManageSettings: boolean; isAdmin: boolean; isStaff: boolean }
) {
  const canManageShared = permissions.isAdmin || permissions.isStaff
  if (!role.organizationId) return canManageShared
  return role.organizationId === permissions.organizationId && (permissions.canManageSettings || canManageShared)
}
