import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  ADMIN_OFFLINE_MAX_AGE,
  deserializeSnapshot,
  OfflineSnapshots,
  type SnapshotDocument,
  type SnapshotStorage,
  serializeSnapshot
} from './offline-snapshots'

function memoryStorage(): SnapshotStorage {
  let value: SnapshotDocument | null = null
  return {
    read: async () => value,
    update: async (change) => {
      value = change(value)
      return value
    }
  }
}

const identity = (id: string, now: number, lifetime = ADMIN_OFFLINE_MAX_AGE * 2) => ({
  identityId: id,
  displayName: `User ${id}`,
  email: `${id}@example.test`,
  verifiedAt: now,
  sessionExpiresAt: now + lifetime
})

test('only opened whitelisted reads persist after a verified identity, with bigint intact', async () => {
  const store = new OfflineSnapshots(memoryStorage(), () => 1000)
  assert.equal(await store.save(store.lease(), 'events.v1.CategoriesService/ListCategories', {}, {}), false)
  await store.verify(identity('a', 1000))
  assert.equal(await store.save(store.lease(), 'events.v1.CategoriesService/CreateCategory', {}, {}), false)
  assert.equal(await store.save(store.lease(), 'events.v1.EventAttendanceService/GetCheckInInfo', {}, {}), false)
  const data = { categories: [{ id: 9007199254740993n, name: 'Academic' }] }
  assert.equal(await store.save(store.lease(), 'events.v1.CategoriesService/ListCategories', { page: 1 }, data), true)
  const saved = await store.read('events.v1.CategoriesService/ListCategories', { page: 1 })
  assert.deepEqual(saved?.data, data)
  assert.equal(await store.read('events.v1.CategoriesService/ListCategories', { page: 2 }), null)
})

test('snapshots expire at the earlier of 24 hours or the last verified session expiry', async () => {
  let now = 1000
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => now)
  await store.verify(identity('a', now, 2000))
  await store.save(
    store.lease(),
    'users.v1.UsersService/GetMe',
    {},
    { user: { name: 'A' }, permissions: { isAdmin: true } }
  )
  now = 3000
  assert.equal(await store.restore(), null)
  assert.equal(await storage.read(), null)
  await store.verify(identity('a', now))
  await store.save(store.lease(), 'users.v1.UsersService/GetMe', {}, {})
  now += ADMIN_OFFLINE_MAX_AGE
  assert.equal(await store.restore(), null)
})

test('switching accounts destroys private reads and rejects a late response from the old identity', async () => {
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('a', 1000))
  const oldLease = store.lease()
  await store.save(oldLease, 'users.v1.UsersService/GetMe', {}, { user: { name: 'A' } })
  await store.verify(identity('b', 1000))
  assert.equal(await store.save(oldLease, 'users.v1.UsersService/GetMe', {}, { user: { name: 'A' } }), false)
  assert.equal(await store.read('users.v1.UsersService/GetMe', {}), null)
  assert.equal((await storage.read())?.identity.identityId, 'b')
})

test('logout invalidates leases before deletion and a fresh cold offline launch has no identity', async () => {
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('a', 1000))
  const oldLease = store.lease()
  await store.clear()
  assert.equal(store.isCurrent(oldLease), false)
  assert.equal(await store.save(oldLease, 'users.v1.UsersService/GetMe', {}, { user: { name: 'A' } }), false)
  const coldStore = new OfflineSnapshots(storage, () => 1000)
  assert.equal(await coldStore.restore(), null)
})

test('a login reset fences the same identity old reads and old logout deletion', async () => {
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('a', 1000), undefined, 'old-login')
  const oldLease = store.lease()
  await store.verify(identity('a', 1000), undefined, 'new-login')
  await store.save(store.lease(), 'users.v1.UsersService/GetMe', {}, { user: { name: 'New session' } })
  assert.equal(store.isCurrent(oldLease), false)
  assert.equal(await store.save(oldLease, 'users.v1.UsersService/GetMe', {}, { user: { name: 'Old session' } }), false)
  await store.clearIdentity('a', 'old-login')
  assert.equal((await storage.read())?.identity.identityId, 'a')
  assert.deepEqual((await store.read('users.v1.UsersService/GetMe', {}))?.data, { user: { name: 'New session' } })
})

test('storage denial permits online identity display without making a durable offline snapshot', async () => {
  const storage: SnapshotStorage = {
    read: async () => {
      throw new Error('SecurityError')
    },
    update: async () => {
      throw new Error('QuotaExceededError')
    }
  }
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('a', 1000))
  assert.equal(store.identity?.identityId, 'a')
  assert.equal(store.storageAvailable, false)
  assert.equal(await store.save(store.lease(), 'users.v1.UsersService/GetMe', {}, {}), false)
  assert.equal(await new OfflineSnapshots(storage, () => 1000).restore(), null)
})

test('serialization never includes credentials, OAuth state or session tokens', () => {
  const encoded = serializeSnapshot({
    id: 3n,
    token: 'secret',
    csrfToken: 'secret',
    session: { active: true },
    authorization: 'secret',
    password: 'secret',
    nested: { access_token: 'secret', refreshToken: 'secret', title: 'Allowed' }
  })
  assert.doesNotMatch(encoded, /secret|active/)
  assert.deepEqual(deserializeSnapshot(encoded), { id: 3n, nested: { title: 'Allowed' } })
})

test('a live session without a finite expiry cannot create a durable offline identity', async () => {
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('a', 1000, Number.POSITIVE_INFINITY))
  assert.equal(store.identity?.identityId, 'a')
  assert.equal(await store.save(store.lease(), 'users.v1.UsersService/GetMe', {}, {}), false)
  assert.equal(await storage.read(), null)
})

test('a delayed cross-app revocation of A does not delete the newer B snapshot', async () => {
  const storage = memoryStorage()
  const store = new OfflineSnapshots(storage, () => 1000)
  await store.verify(identity('b', 1000))
  await store.save(store.lease(), 'users.v1.UsersService/GetMe', {}, { user: { name: 'B' } })
  await store.clearIdentity('a')
  assert.equal(store.identity?.identityId, 'b')
  assert.equal((await storage.read())?.identity.identityId, 'b')
  await store.clearIdentity('b')
  assert.equal(store.identity, null)
  assert.equal(await storage.read(), null)
})

test('expired A read cleanup preserves B committed by another tab before the cleanup transaction', async () => {
  const expired: SnapshotDocument = { version: 1, epoch: 'a-old', identity: identity('a', 1000, 1), reads: [] }
  let current: SnapshotDocument | null = { version: 1, epoch: 'b-new', identity: identity('b', 2000), reads: [] }
  const storage: SnapshotStorage = { read: async () => expired, update: async (change) => (current = change(current)) }
  assert.equal(await new OfflineSnapshots(storage, () => 2000).restore(), null)
  assert.equal(current?.identity.identityId, 'b')
})

test('a missing initial snapshot never deletes a snapshot committed by another tab', async () => {
  let current: SnapshotDocument | null = { version: 1, epoch: 'b-new', identity: identity('b', 2000), reads: [] }
  const storage: SnapshotStorage = { read: async () => null, update: async (change) => (current = change(current)) }
  assert.equal(await new OfflineSnapshots(storage, () => 2000).restore(), null)
  assert.equal(current?.identity.identityId, 'b')
})

test('an initial response without an owner cannot replace a newer B snapshot with A', async () => {
  let current: SnapshotDocument | null = { version: 1, epoch: 'b-new', identity: identity('b', 2000), reads: [] }
  const storage: SnapshotStorage = { read: async () => current, update: async (change) => (current = change(current)) }
  const store = new OfflineSnapshots(storage, () => 2000)
  assert.equal(await store.verify(identity('a', 2000), null), false)
  assert.equal(current?.identity.identityId, 'b')
  assert.equal(store.identity, null)
})

test('a late storage rejection from restore cannot invalidate a newer verified identity', async () => {
  let rejectRead: ((error: Error) => void) | undefined
  const pendingRead = new Promise<SnapshotDocument | null>((_resolve, reject) => {
    rejectRead = reject
  })
  let current: SnapshotDocument | null = null
  const storage: SnapshotStorage = {
    read: async () => pendingRead,
    update: async (change) => (current = change(current))
  }
  const store = new OfflineSnapshots(storage, () => 2000)
  const restoring = store.restore()
  await store.verify(identity('b', 2000))
  rejectRead?.(new Error('Storage was temporarily unavailable'))
  assert.equal(await restoring, null)
  assert.equal(store.identity?.identityId, 'b')
  assert.equal(store.storageAvailable, true)
})
