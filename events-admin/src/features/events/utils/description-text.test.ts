import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getEventDescriptionText } from './description-text'

test('event list descriptions show editor text and decode stored plain text', () => {
  assert.equal(getEventDescriptionText('[{"type":"p","children":[{"text":"Campus workshop"}]}]'), 'Campus workshop')
  assert.equal(getEventDescriptionText('"Campus workshop"'), 'Campus workshop')
  assert.equal(getEventDescriptionText('Legacy description'), 'Legacy description')
  assert.equal(getEventDescriptionText('[{"type":"p","children":[{"text":""}]}]'), '')
  assert.equal(getEventDescriptionText(null), '')
})
