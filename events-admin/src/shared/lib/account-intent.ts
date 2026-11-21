// Keep this browser protocol identical in map and events-admin: both scopes share this database.
export const ACCOUNT_INTENT_DATABASE = 'studyverse-account-intents-v1'
const initialGeneration = 'initial'

export interface AccountIntentRecord {
  generation: string
  pendingIdentityIds: string[]
}

export interface AccountIntentStorage {
  read: () => Promise<AccountIntentRecord | null>
  update: (change: (current: AccountIntentRecord | null) => AccountIntentRecord) => Promise<AccountIntentRecord>
}

function initialRecord(): AccountIntentRecord {
  return { generation: initialGeneration, pendingIdentityIds: [] }
}

function newerGeneration(current: string) {
  const previousTime = Number(current.split(':')[0]) || 0
  return `${Math.max(Date.now(), previousTime + 1)}:${crypto.randomUUID()}`
}

function compareGenerations(left: string, right: string) {
  if (left === right) return 0
  if (left === initialGeneration) return -1
  if (right === initialGeneration) return 1
  const time = Number(left.split(':')[0]) - Number(right.split(':')[0])
  return time || left.localeCompare(right)
}

export class AccountIntents {
  readonly pendingIdentities = new Set<string>()
  storageAvailable = true
  private generation = initialGeneration
  private revision = 0
  private readonly storage: AccountIntentStorage

  constructor(storage: AccountIntentStorage) {
    this.storage = storage
  }

  currentGeneration() {
    return this.generation
  }

  isBlocked(identityId: string, persisted: string | null) {
    return (
      this.pendingIdentities.has('true') ||
      this.pendingIdentities.has(identityId) ||
      (this.generation === initialGeneration && (persisted === 'true' || persisted === identityId))
    )
  }

  private adopt(record: AccountIntentRecord) {
    if (compareGenerations(record.generation, this.generation) < 0) return
    if (record.generation !== this.generation) this.pendingIdentities.clear()
    this.generation = record.generation
    for (const identityId of record.pendingIdentityIds) this.pendingIdentities.add(identityId)
  }

  async restore() {
    const revision = this.revision
    try {
      const record = await this.storage.read()
      this.storageAvailable = true
      if (revision === this.revision && record) this.adopt(record)
    } catch {
      this.storageAvailable = false
      // Durable storage is optional; retain the local intent when it cannot be read.
    }
    return this.generation
  }

  async remember(identityId: string, generation = this.generation) {
    if (generation !== this.generation) return false
    this.pendingIdentities.add(identityId)
    const revision = ++this.revision
    let accepted = true
    try {
      const record = await this.storage.update((stored) => {
        const current = stored ?? initialRecord()
        if (current.generation !== generation) {
          accepted = false
          return current
        }
        return {
          generation,
          pendingIdentityIds: [...new Set([...current.pendingIdentityIds, identityId])]
        }
      })
      this.storageAvailable = true
      if (revision === this.revision) this.adopt(record)
    } catch {
      this.storageAvailable = false
      // Keep local sign-out effective even if neither durable storage nor localStorage can be written.
    }
    return accepted && generation === this.generation
  }

  async canResume(identityId: string, checkpoint: string, persisted: string | null) {
    await this.restore()
    return checkpoint === this.generation && !this.isBlocked(identityId, persisted)
  }

  async reset() {
    const revision = ++this.revision
    const generation = newerGeneration(this.generation)
    this.generation = generation
    this.pendingIdentities.clear()
    try {
      const record = await this.storage.update((current) =>
        current && compareGenerations(current.generation, generation) >= 0
          ? current
          : { generation, pendingIdentityIds: [] }
      )
      this.storageAvailable = true
      if (revision === this.revision) this.adopt(record)
    } catch {
      this.storageAvailable = false
      // Explicit sign-in can still proceed with local and broadcast intent reset.
    }
    return this.generation
  }

  async acceptReset(generation: string) {
    const revision = this.revision
    try {
      const record = await this.storage.read()
      this.storageAvailable = true
      if (revision !== this.revision || record?.generation !== generation) return false
      const changed = this.generation !== generation
      this.adopt(record)
      return changed
    } catch {
      this.storageAvailable = false
      if (revision !== this.revision || compareGenerations(generation, this.generation) <= 0) return false
      this.revision += 1
      this.generation = generation
      this.pendingIdentities.clear()
      return true
    }
  }

  async acceptStorageReset() {
    const previous = this.generation
    await this.restore()
    if (this.generation !== previous) return true
    // New clients carry a durable generation and send an explicit reset message.
    // A delayed legacy storage event must not clear a logout made after that reset.
    if (this.storageAvailable && this.generation !== initialGeneration) return false
    await this.reset()
    return true
  }
}

let database: Promise<IDBDatabase> | undefined

function openDatabase() {
  if (!database)
    database = new Promise<IDBDatabase>((resolve, reject) => {
      const request = indexedDB.open(ACCOUNT_INTENT_DATABASE, 1)
      request.onupgradeneeded = () => request.result.createObjectStore('intents')
      request.onsuccess = () => {
        request.result.onversionchange = () => {
          request.result.close()
          database = undefined
        }
        resolve(request.result)
      }
      request.onerror = () => reject(request.error)
      request.onblocked = () => reject(new Error('Account intent storage blocked'))
    })
  return database
}

const storage: AccountIntentStorage = {
  read: async () => {
    const db = await openDatabase()
    return new Promise((resolve, reject) => {
      const transaction = db.transaction('intents', 'readonly')
      const request = transaction.objectStore('intents').get('active')
      request.onsuccess = () => resolve((request.result as AccountIntentRecord | undefined) ?? null)
      request.onerror = () => reject(request.error)
      transaction.onabort = () => reject(transaction.error)
    })
  },
  update: async (change) => {
    const db = await openDatabase()
    return new Promise((resolve, reject) => {
      const transaction = db.transaction('intents', 'readwrite')
      const store = transaction.objectStore('intents')
      const request = store.get('active')
      let record = initialRecord()
      request.onsuccess = () => {
        record = change((request.result as AccountIntentRecord | undefined) ?? null)
        store.put(record, 'active')
      }
      transaction.oncomplete = () => resolve(record)
      transaction.onerror = () => reject(transaction.error)
      transaction.onabort = () => reject(transaction.error)
    })
  }
}

export const accountIntents = new AccountIntents(storage)
