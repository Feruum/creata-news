import assert from 'node:assert/strict'
import { test } from 'node:test'
import { OrganizationStatus } from '@studyverse/proto'
import { EMPTY_PLATE_VALUE } from '@/shared/lib/plate-utils'
import {
  getOrganizationCreateInput,
  getOrganizationFormValues,
  getOrganizationUpdateInput,
  hasOrganizationFormChanges,
  organizationFormSchema
} from './organization-form'

test('organization descriptions count visible Plate text rather than serialized formatting', () => {
  const values = getOrganizationFormValues()
  const description = JSON.stringify([{ type: 'p', children: [{ text: 'x'.repeat(2000), bold: true }] }])
  assert.ok(description.length > 2000)
  assert.equal(
    organizationFormSchema.safeParse({ ...values, title: 'Chess club', organizationTypeId: 1, description }).success,
    true
  )
  assert.equal(
    organizationFormSchema.safeParse({
      ...values,
      title: 'Chess club',
      organizationTypeId: 1,
      description: JSON.stringify([{ type: 'p', children: [{ text: 'x'.repeat(2001) }] }])
    }).success,
    false
  )
})

test('editor-generated element IDs and JSON key ordering do not create an unsaved draft', () => {
  const initial = {
    ...getOrganizationFormValues(),
    description: JSON.stringify([{ type: 'p', children: [{ text: 'Club description' }] }])
  }
  const normalized = {
    ...initial,
    description: JSON.stringify([{ id: 'plate-block', children: [{ text: 'Club description' }], type: 'p' }])
  }
  assert.equal(hasOrganizationFormChanges(normalized, initial), false)
  assert.equal(hasOrganizationFormChanges({ ...normalized, title: 'New name' }, initial), true)
  assert.equal(hasOrganizationFormChanges({ ...normalized, removeImage: true }, initial), true)
})

test('rich-text changes stay dirty and restoring the original content clears the draft', () => {
  const initial = {
    ...getOrganizationFormValues(),
    description: JSON.stringify([{ type: 'p', children: [{ text: 'Club description' }] }])
  }
  for (const description of [
    [{ type: 'p', id: 'plate-block', children: [{ text: 'Changed description' }] }],
    [{ type: 'p', id: 'plate-block', children: [{ text: 'Club description', bold: true }] }],
    [{ type: 'h1', id: 'plate-block', children: [{ text: 'Club description' }] }]
  ]) {
    assert.equal(hasOrganizationFormChanges({ ...initial, description: JSON.stringify(description) }, initial), true)
  }
  const restored = [{ type: 'p', id: 'new-block-id', children: [{ text: 'Club description' }] }]
  assert.equal(hasOrganizationFormChanges({ ...initial, description: JSON.stringify(restored) }, initial), false)
})

test('organization edits target the existing organization and preserve an unchanged title and image', () => {
  const values = { ...getOrganizationFormValues(), title: 'Chess club', organizationTypeId: 4 }
  const input = getOrganizationUpdateInput(values, { id: 27, title: 'Chess club' })
  assert.equal(input.id, 27)
  assert.equal(input.title, undefined)
  assert.equal(input.imageUrl, undefined)
  assert.equal(input.organizationTypeId, 4)
  assert.equal(input.status, OrganizationStatus.ACTIVE)
})

test('organization edits explicitly clear removed image, description and social links', () => {
  const values = { ...getOrganizationFormValues(), title: 'Renamed club', organizationTypeId: 4, removeImage: true }
  const input = getOrganizationUpdateInput(values, { id: 27, title: 'Old name' })
  assert.equal(input.title, 'Renamed club')
  assert.equal(input.imageUrl, '')
  assert.deepEqual(JSON.parse(input.description), EMPTY_PLATE_VALUE)
  for (const field of [
    'website',
    'instagram',
    'telegramChannel',
    'telegramChat',
    'youtube',
    'tiktok',
    'linkedin'
  ] as const) {
    assert.equal(input[field], '')
  }
})

test('an uploaded image replaces the existing image without adding edit-only flags to requests', () => {
  const values = { ...getOrganizationFormValues(), title: 'Chess club', organizationTypeId: 4 }
  const input = getOrganizationCreateInput(values, 'https://images.example.test/new.png')
  assert.equal(input.imageUrl, 'https://images.example.test/new.png')
  assert.equal(input.title, 'Chess club')
  assert.equal(Object.hasOwn(input, 'removeImage'), false)
  assert.equal(Object.hasOwn(input, 'coverImage'), false)
})

test('cleared descriptions remain valid JSON documents for the backend JSONB column', () => {
  const values = {
    ...getOrganizationFormValues(),
    title: 'Chess club',
    organizationTypeId: 4,
    description: JSON.stringify([{ type: 'p', children: [{ text: '' }] }])
  }
  for (const description of ['', undefined, JSON.stringify(EMPTY_PLATE_VALUE)]) {
    const clearedValues = { ...values, description }
    assert.deepEqual(JSON.parse(getOrganizationCreateInput(clearedValues).description), EMPTY_PLATE_VALUE)
    assert.deepEqual(
      JSON.parse(getOrganizationUpdateInput(clearedValues, { id: 27, title: 'Chess club' }).description),
      EMPTY_PLATE_VALUE
    )
  }
})

test('plain and serialized descriptions are safely encoded for raw JSON storage on resave', () => {
  const description = [{ type: 'p', children: [{ text: 'Chess practice', bold: true }] }]
  const values = { ...getOrganizationFormValues(), title: 'Chess club', organizationTypeId: 4 }
  assert.deepEqual(JSON.parse(getOrganizationCreateInput({ ...values, description: 'Chess practice' }).description), [
    { type: 'p', children: [{ text: 'Chess practice' }] }
  ])
  assert.deepEqual(
    JSON.parse(
      getOrganizationUpdateInput(
        { ...values, description: JSON.stringify(description) },
        { id: 27, title: 'Chess club' }
      ).description
    ),
    description
  )
})

test('organization images enforce the backend formats and 5 MB limit', () => {
  const values = { ...getOrganizationFormValues(), title: 'Chess club', organizationTypeId: 1 }
  for (const type of ['image/jpeg', 'image/png', 'image/webp', 'image/gif']) {
    assert.equal(
      organizationFormSchema.safeParse({ ...values, coverImage: new File(['x'], 'logo', { type }) }).success,
      true
    )
  }
  for (const coverImage of [
    new File(['<svg />'], 'logo.svg', { type: 'image/svg+xml' }),
    new File([], 'empty.png', { type: 'image/png' }),
    new File([new Uint8Array(5 * 1024 * 1024 + 1)], 'large.png', { type: 'image/png' })
  ]) {
    assert.equal(organizationFormSchema.safeParse({ ...values, coverImage }).success, false)
  }
})
