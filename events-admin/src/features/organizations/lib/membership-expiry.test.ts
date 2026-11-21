import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { membershipExpiryInput, membershipExpiryValue } from './membership-expiry'

describe('membership expiry', () => {
  test('an empty value removes the expiry', () => {
    assert.equal(membershipExpiryValue(''), '')
    assert.equal(membershipExpiryInput(undefined), '')
  })
  test('editing a saved end date preserves its instant in the local datetime input', () => {
    const saved = '2035-06-01T12:30:00.000Z'
    assert.equal(membershipExpiryValue(membershipExpiryInput(saved)), saved)
  })
  test('rejects invalid dates and dates that already expired', () => {
    assert.throws(() => membershipExpiryValue('not a date'), /future/)
    assert.throws(() => membershipExpiryValue('2000-01-01T12:00'), /future/)
  })
})
