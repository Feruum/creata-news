import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { ACCOUNT_REVOCATION_KEY, resetLocalLogout } from './account-revocation'
import {
  clearPrivateSnapshots,
  getOfflineState,
  offlineSnapshots,
  onPrivateCacheInvalidated,
  verifyOfflineIdentity
} from './offline-runtime'
import type { SnapshotOwner } from './offline-snapshots'

const firstIdentity = {
  identityId: 'a',
  displayName: 'Account A',
  email: 'a@example.test',
  verifiedAt: Date.now(),
  sessionExpiresAt: Date.now() + 60_000
}
const renewedIdentity = {
  ...firstIdentity,
  verifiedAt: firstIdentity.verifiedAt + 1000,
  sessionExpiresAt: firstIdentity.sessionExpiresAt + 1000
}

function captureOwner(): SnapshotOwner {
  const lease = offlineSnapshots.lease()
  assert.ok(lease)
  assert.ok(offlineSnapshots.identity)
  return { ...lease, verifiedAt: offlineSnapshots.identity.verifiedAt }
}

function captureMessages() {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const storageDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  const messages: string[] = []
  const storage: Storage = {
    length: 0,
    key: () => null,
    getItem: () => null,
    setItem: (key, value) => {
      if (key === ACCOUNT_REVOCATION_KEY) messages.push(value)
    },
    removeItem: () => undefined,
    clear: () => undefined
  }
  Object.defineProperty(globalThis, 'window', { value: { localStorage: storage }, configurable: true })
  Object.defineProperty(globalThis, 'localStorage', { value: storage, configurable: true })
  return {
    messages,
    restore: () => {
      if (descriptor) Object.defineProperty(globalThis, 'window', descriptor)
      else Reflect.deleteProperty(globalThis, 'window')
      if (storageDescriptor) Object.defineProperty(globalThis, 'localStorage', storageDescriptor)
      else Reflect.deleteProperty(globalThis, 'localStorage')
    }
  }
}

afterEach(async () => {
  await clearPrivateSnapshots('logout')
  resetLocalLogout()
})

test('stale automatic expiry cannot broadcast or invalidate a renewed session of the same identity', async () => {
  await verifyOfflineIdentity(firstIdentity)
  const oldOwner = captureOwner()
  await verifyOfflineIdentity(renewedIdentity)
  const recorder = captureMessages()
  try {
    await clearPrivateSnapshots('session-expired', undefined, oldOwner)
    assert.equal(getOfflineState().identity?.verifiedAt, renewedIdentity.verifiedAt)
    assert.equal(getOfflineState().mode, 'online')
    assert.equal(recorder.messages.length, 0)
  } finally {
    recorder.restore()
  }
})

test('renewal during a pending automatic clear cannot receive an old expiry broadcast', async () => {
  await verifyOfflineIdentity(firstIdentity)
  const oldOwner = captureOwner()
  let release: ((removed: boolean) => void) | undefined
  const pending = new Promise<boolean>((resolve) => {
    release = resolve
  })
  const originalClear = offlineSnapshots.clearOwner
  offlineSnapshots.clearOwner = async () => pending
  const recorder = captureMessages()
  try {
    const clearing = clearPrivateSnapshots('session-expired', undefined, oldOwner)
    await verifyOfflineIdentity(renewedIdentity)
    release?.(false)
    await clearing
    assert.equal(getOfflineState().identity?.verifiedAt, renewedIdentity.verifiedAt)
    assert.equal(getOfflineState().mode, 'online')
    assert.equal(recorder.messages.length, 0)
  } finally {
    offlineSnapshots.clearOwner = originalClear
    recorder.restore()
  }
})

test('a conditional delete rejected by newer persisted metadata cannot revoke another tab of the same identity', async () => {
  await verifyOfflineIdentity(firstIdentity)
  const oldOwner = captureOwner()
  const originalClear = offlineSnapshots.clearOwner
  const originalStorageAvailable = offlineSnapshots.storageAvailable
  offlineSnapshots.storageAvailable = true
  offlineSnapshots.clearOwner = async () => false
  const recorder = captureMessages()
  try {
    await clearPrivateSnapshots('session-expired', undefined, oldOwner)
    assert.equal(getOfflineState().mode, 'unverified')
    assert.equal(recorder.messages.length, 0)
  } finally {
    offlineSnapshots.clearOwner = originalClear
    offlineSnapshots.storageAvailable = originalStorageAvailable
    recorder.restore()
  }
})

test('explicit logout invalidates a renewed matching identity even when its snapshot cannot be deleted', async () => {
  await verifyOfflineIdentity(firstIdentity)
  const oldOwner = captureOwner()
  await verifyOfflineIdentity(renewedIdentity)
  const recorder = captureMessages()
  const invalidations: string[] = []
  const remove = onPrivateCacheInvalidated((event) => invalidations.push(event.reason))
  try {
    await clearPrivateSnapshots('logout', undefined, oldOwner)
    assert.equal(getOfflineState().identity, null)
    assert.equal(getOfflineState().mode, 'unverified')
    assert.deepEqual(invalidations, ['logout'])
    assert.equal(recorder.messages.length, 1)
  } finally {
    remove()
    recorder.restore()
  }
})
