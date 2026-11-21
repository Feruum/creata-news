import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  isAccountRevocation,
  noteLocalLogout,
  resetLocalLogout,
  shouldRevokeIdentity,
  wasLocallyLoggedOut
} from './account-revocation'

const revoked = {
  kind: 'account-revoked' as const,
  source: 'map' as const,
  reason: 'account-change' as const,
  previousIdentityId: 'a',
  nonce: '12345'
}

test('cross-app revocation only targets the previous account, even when received late', () => {
  assert.equal(isAccountRevocation(revoked), true)
  assert.equal(shouldRevokeIdentity('a', revoked), true)
  assert.equal(shouldRevokeIdentity('b', revoked), false)
  assert.equal(shouldRevokeIdentity(null, revoked), false)
})

test('unknown broadcasts cannot delete private snapshots', () => {
  assert.equal(isAccountRevocation({ kind: 'account-revoked', previousIdentityId: 'a' }), false)
  assert.equal(isAccountRevocation({ ...revoked, source: 'telemetry' }), false)
  assert.equal(isAccountRevocation({ ...revoked, reason: 'refresh' }), false)
})

test('readable storage cannot discard local logout intent when quota rejects its write', () => {
  const windowDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const storageDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const saved = new Map<string, string>()
  const storage: Storage = {
    get length() {
      return saved.size
    },
    key: (index) => [...saved.keys()][index] ?? null,
    getItem: (key) => saved.get(key) ?? null,
    setItem: () => {
      throw new DOMException('Storage quota exceeded', 'QuotaExceededError')
    },
    removeItem: (key) => {
      saved.delete(key)
    },
    clear: () => saved.clear()
  }
  Object.defineProperty(globalThis, 'window', { value: { localStorage: storage }, configurable: true })
  Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })
  try {
    resetLocalLogout()
    noteLocalLogout('a')
    assert.equal(storage.getItem('studyverse-logout-pending'), null)
    assert.equal(wasLocallyLoggedOut('a'), true)
    assert.equal(wasLocallyLoggedOut('b'), false)
    saved.set('studyverse-logout-pending', 'b')
    assert.equal(wasLocallyLoggedOut('a'), true)
    noteLocalLogout('b')
    assert.equal(wasLocallyLoggedOut('a'), true)
    assert.equal(wasLocallyLoggedOut('b'), true)
    resetLocalLogout()
    assert.equal(wasLocallyLoggedOut('a'), false)
    assert.equal(wasLocallyLoggedOut('b'), false)
  } finally {
    resetLocalLogout()
    if (windowDescriptor) Object.defineProperty(globalThis, 'window', windowDescriptor)
    else Reflect.deleteProperty(globalThis, 'window')
    if (storageDescriptor) Object.defineProperty(globalThis, 'localStorage', storageDescriptor)
    else Reflect.deleteProperty(globalThis, 'localStorage')
  }
})
