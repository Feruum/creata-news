import assert from 'node:assert/strict'
import { test } from 'node:test'
import { type AccountIntentRecord, type AccountIntentStorage, AccountIntents } from './account-intent'

function memoryStorage(): AccountIntentStorage {
  let record: AccountIntentRecord | null = null
  let pending = Promise.resolve()
  return {
    read: async () => {
      await pending
      return record
    },
    update: (change) => {
      const updating = pending.then(() => (record = change(record)))
      pending = updating.then(() => {})
      return updating
    }
  }
}

test('logout intent survives a cold runtime without retaining any private snapshot', async () => {
  const storage = memoryStorage()
  const warm = new AccountIntents(storage)
  await warm.restore()
  await warm.remember('alice')
  const cold = new AccountIntents(storage)
  const checkpoint = await cold.restore()
  assert.equal(await cold.canResume('alice', checkpoint, null), false)
  assert.equal(await cold.canResume('bob', checkpoint, null), true)
  assert.deepEqual(Object.keys((await storage.read()) ?? {}).sort(), ['generation', 'pendingIdentityIds'])
})

test('explicit login from the other scope clears durable intent even when the logged-out app is closed', async () => {
  const storage = memoryStorage()
  const map = new AccountIntents(storage)
  await map.remember('alice')
  const admin = new AccountIntents(storage)
  await admin.restore()
  await admin.reset()
  const coldMap = new AccountIntents(storage)
  const checkpoint = await coldMap.restore()
  assert.equal(await coldMap.canResume('alice', checkpoint, null), true)
})

test('a login reset rejects an earlier session verification and an earlier logout write', async () => {
  const storage = memoryStorage()
  const oldTab = new AccountIntents(storage)
  const oldGeneration = await oldTab.restore()
  const newTab = new AccountIntents(storage)
  await newTab.restore()
  const reset = await newTab.reset()
  assert.equal(await oldTab.remember('alice', oldGeneration), false)
  assert.equal(await oldTab.canResume('alice', oldGeneration, null), false)
  assert.equal(await oldTab.canResume('alice', reset, null), true)
})

test('a delayed reset message cannot clear logout intent created after that reset', async () => {
  const storage = memoryStorage()
  const tab = new AccountIntents(storage)
  const generation = await tab.reset()
  await tab.remember('alice')
  await tab.acceptReset(generation)
  assert.equal(tab.isBlocked('alice', null), true)
})

test('older reset broadcasts cannot replace a newer durable login generation', async () => {
  const storage = memoryStorage()
  const first = new AccountIntents(storage)
  const oldReset = await first.reset()
  const second = new AccountIntents(storage)
  await second.restore()
  const newReset = await second.reset()
  assert.equal(await first.acceptReset(oldReset), false)
  assert.equal(await first.acceptReset(newReset), true)
  assert.equal(first.currentGeneration(), newReset)
})

test('unavailable durable storage keeps local logout and explicit login usable', async () => {
  const storage: AccountIntentStorage = {
    read: async () => {
      throw new Error('Storage blocked')
    },
    update: async () => {
      throw new Error('Storage blocked')
    }
  }
  const tab = new AccountIntents(storage)
  await tab.remember('alice')
  assert.equal(tab.isBlocked('alice', null), true)
  const generation = await tab.reset()
  assert.equal(tab.isBlocked('alice', null), false)
  await tab.remember('alice')
  await tab.acceptReset(generation)
  assert.equal(tab.isBlocked('alice', null), true)
})

test('a delayed storage removal cannot clear a logout made after a durable login reset', async () => {
  const tab = new AccountIntents(memoryStorage())
  await tab.reset()
  await tab.remember('alice')
  assert.equal(await tab.acceptStorageReset(), false)
  assert.equal(tab.isBlocked('alice', null), true)
})

test('a read begun before explicit login cannot restore old logout intent after the reset', async () => {
  let release: ((record: AccountIntentRecord) => void) | undefined
  const reading = new Promise<AccountIntentRecord>((resolve) => {
    release = resolve
  })
  const tab = new AccountIntents({ read: () => reading, update: async (change) => change(null) })
  const restoring = tab.restore()
  const generation = await tab.reset()
  release?.({ generation: 'initial', pendingIdentityIds: ['alice'] })
  await restoring
  assert.equal(tab.currentGeneration(), generation)
  assert.equal(tab.isBlocked('alice', null), false)
})

test('concurrent unrelated logout intents do not supersede each other within the same login generation', async () => {
  const storage = memoryStorage()
  const tab = new AccountIntents(storage)
  const first = tab.remember('alice')
  const second = tab.remember('bob')
  assert.equal(await first, true)
  assert.equal(await second, true)
  assert.deepEqual((await storage.read())?.pendingIdentityIds, ['alice', 'bob'])
})
