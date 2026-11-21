import assert from 'node:assert/strict'
import { test } from 'node:test'
import { canEditRoleTemplate, getRoleUpdateInput } from './role-template-form'

test('removing role parent and member limits sends explicit clear flags', () => {
  const input = getRoleUpdateInput({
    id: 12,
    displayName: 'Member',
    description: '',
    parentTemplateId: '',
    maxHolders: '',
    sortOrder: '5',
    isRequired: false,
    isActive: false,
    capabilities: []
  })
  assert.equal(input.clearParentTemplate, true)
  assert.equal(input.parentTemplateId, undefined)
  assert.equal(input.clearMaxHolders, true)
  assert.equal(input.maxHolders, undefined)
  assert.equal(input.description, '')
  assert.equal(input.sortOrder, 5)
  assert.equal(input.isActive, false)
})

test('role limits and ordering reject invalid integers before requesting a mutation', () => {
  const values = {
    id: 12,
    displayName: 'Member',
    description: '',
    parentTemplateId: '3',
    maxHolders: '2',
    sortOrder: '0',
    isRequired: true,
    isActive: true,
    capabilities: []
  }
  const input = getRoleUpdateInput(values)
  assert.equal(input.clearParentTemplate, false)
  assert.equal(input.parentTemplateId, 3)
  assert.equal(input.clearMaxHolders, false)
  assert.equal(input.maxHolders, 2)
  for (const maxHolders of ['0', '-1', '1.5', 'abc']) {
    assert.throws(() => getRoleUpdateInput({ ...values, maxHolders }), /Max members/)
  }
  for (const sortOrder of ['1.5', 'abc', '2147483648']) {
    assert.throws(() => getRoleUpdateInput({ ...values, sortOrder }), /Sort order/)
  }
})

test('club settings managers may edit their local roles but shared templates require staff or admin', () => {
  const manager = { organizationId: 7, canManageSettings: true, isAdmin: false, isStaff: false }
  assert.equal(canEditRoleTemplate({ organizationId: 7 }, manager), true)
  assert.equal(canEditRoleTemplate({ organizationId: 8 }, manager), false)
  assert.equal(canEditRoleTemplate({ organizationId: undefined }, manager), false)
  assert.equal(canEditRoleTemplate({ organizationId: undefined }, { ...manager, isStaff: true }), true)
  assert.equal(canEditRoleTemplate({ organizationId: undefined }, { ...manager, isAdmin: true }), true)
  assert.equal(canEditRoleTemplate({ organizationId: 7 }, { ...manager, canManageSettings: false }), false)
})
