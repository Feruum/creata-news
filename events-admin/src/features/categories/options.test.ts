import assert from 'node:assert/strict'
import { test } from 'node:test'
import { buildCategoryOptions } from './options'
import { categoryPresets, existingCategoryNames } from './presets'

test('unused real categories are available as event filters with their server IDs', () => {
  const options = buildCategoryOptions(
    [{ id: 23, name: 'Music', color: '#8B5CF6' }],
    [{ id: 3, name: 'Career', color: '#FCB900' }]
  )
  assert.deepEqual(
    options.map(({ value, label }) => ({ value, label })),
    [
      { value: 3, label: 'Career' },
      { value: 23, label: 'Music' }
    ]
  )
})

test('catalog names and colors take precedence while event-only categories remain reachable', () => {
  assert.deepEqual(
    buildCategoryOptions(
      [{ id: 3, name: 'Career', color: '#FCB900' }],
      [
        { id: 3, name: 'Old name', color: '#000000' },
        { id: 8, name: 'Science', color: '' }
      ]
    ),
    [
      { value: 3, label: 'Career', color: '#FCB900' },
      { value: 8, label: 'Science', color: '#6B7280' }
    ]
  )
})

test('suggestions recognize existing names regardless of case or surrounding spaces', () => {
  assert.equal(existingCategoryNames([{ name: '  mUsIc ' }]).has('music'), true)
  assert.equal(new Set(categoryPresets.map(({ name }) => name.toLowerCase())).size, categoryPresets.length)
  assert.ok(categoryPresets.some(({ name }) => name === 'Music'))
  assert.ok(categoryPresets.every(({ color }) => /^#[0-9A-F]{6}$/.test(color)))
})
