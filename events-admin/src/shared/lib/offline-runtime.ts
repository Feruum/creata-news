import {
  ACCOUNT_CHANNEL,
  ACCOUNT_REVOCATION_KEY,
  type AccountRevocation,
  applyAccountLoginReset,
  canResumeAccount,
  currentAccountIntent,
  isAccountRevocation,
  noteLocalLogout,
  parseAccountLoginReset,
  type RevocationReason,
  shouldRevokeIdentity,
  wasLocallyLoggedOut
} from './account-revocation'
import { writeBrowserSetting } from './browser-storage'
import {
  type IdentityLease,
  type OfflineIdentity,
  OfflineSnapshots,
  type SnapshotDocument,
  type SnapshotOwner,
  type SnapshotStorage,
  snapshotExpiresAt
} from './offline-snapshots'

const basePath = import.meta.env?.BASE_URL ?? '/'
const databaseName = `studyverse-admin-offline-v1:${basePath}`
const listeners = new Set<() => void>()
interface PrivateCacheInvalidation {
  reason: RevocationReason
  local: boolean
}
const invalidationListeners = new Set<(event: PrivateCacheInvalidation) => void>()
let transition = 0
let database: Promise<IDBDatabase> | undefined

function openDatabase(): Promise<IDBDatabase> {
  if (database) return database
  database = new Promise((resolve, reject) => {
    if (typeof indexedDB === 'undefined') return reject(new Error('Offline storage is unavailable'))
    const request = indexedDB.open(databaseName, 1)
    request.onupgradeneeded = () => request.result.createObjectStore('snapshots')
    request.onerror = () => reject(request.error)
    request.onblocked = () => reject(new Error('Offline storage is blocked'))
    request.onsuccess = () => {
      request.result.onversionchange = () => request.result.close()
      resolve(request.result)
    }
  })
  return database
}

const storage: SnapshotStorage = {
  read: async () => {
    const db = await openDatabase()
    return new Promise<SnapshotDocument | null>((resolve, reject) => {
      const transaction = db.transaction('snapshots', 'readonly')
      const request = transaction.objectStore('snapshots').get('active')
      request.onsuccess = () => resolve((request.result as SnapshotDocument | undefined) ?? null)
      request.onerror = () => reject(request.error)
      transaction.onabort = () => reject(transaction.error)
    })
  },
  update: async (change) => {
    const db = await openDatabase()
    return new Promise<SnapshotDocument | null>((resolve, reject) => {
      const transaction = db.transaction('snapshots', 'readwrite')
      const store = transaction.objectStore('snapshots')
      const request = store.get('active')
      let updated: SnapshotDocument | null = null
      request.onsuccess = () => {
        updated = change((request.result as SnapshotDocument | undefined) ?? null)
        if (updated) store.put(updated, 'active')
        else store.delete('active')
      }
      transaction.oncomplete = () => resolve(updated)
      transaction.onerror = () => reject(transaction.error)
      transaction.onabort = () => reject(transaction.error)
    })
  }
}

export const offlineSnapshots = new OfflineSnapshots(storage)
export type OfflineMode = 'unverified' | 'online' | 'offline-readonly'
export interface OfflineState {
  mode: OfflineMode
  identity: OfflineIdentity | null
  storageAvailable: boolean
  lastUpdatedAt: number | null
}

let state: OfflineState = { mode: 'unverified', identity: null, storageAvailable: true, lastUpdatedAt: null }
let verification: (() => Promise<boolean>) | undefined
let verificationNeeded = true
let expirationTimer: ReturnType<typeof setTimeout> | undefined
const channel =
  typeof window !== 'undefined' && typeof BroadcastChannel !== 'undefined'
    ? new BroadcastChannel(ACCOUNT_CHANNEL)
    : null

function publish(change: Partial<OfflineState> = {}) {
  state = {
    ...state,
    identity: offlineSnapshots.identity,
    storageAvailable: offlineSnapshots.storageAvailable,
    ...change
  }
  listeners.forEach((listener) => {
    listener()
  })
}

export function subscribeOffline(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getOfflineState(): OfflineState {
  return state
}

export function onPrivateCacheInvalidated(listener: (event: PrivateCacheInvalidation) => void): () => void {
  invalidationListeners.add(listener)
  return () => invalidationListeners.delete(listener)
}

function invalidateMemory(reason: RevocationReason, local: boolean) {
  offlineSnapshots.invalidate()
  verificationNeeded = true
  clearTimeout(expirationTimer)
  publish({ mode: 'unverified', lastUpdatedAt: null })
  invalidationListeners.forEach((listener) => {
    listener({ reason, local })
  })
}

const receivedNonces = new Set<string>()
async function receiveRevocation(value: unknown): Promise<void> {
  const reset = parseAccountLoginReset(value)
  if (reset) {
    if (await applyAccountLoginReset(reset)) {
      transition += 1
      invalidateMemory('session-expired', false)
    }
    return
  }
  if (!isAccountRevocation(value) || receivedNonces.has(value.nonce)) return
  if (value.generation && value.generation !== currentAccountIntent()) return
  const intentGeneration = value.generation ?? currentAccountIntent()
  if (value.reason === 'logout' && !(await noteLocalLogout(value.previousIdentityId, intentGeneration))) return
  receivedNonces.add(value.nonce)
  if (receivedNonces.size > 80) receivedNonces.delete(receivedNonces.values().next().value ?? '')
  const matches = shouldRevokeIdentity(offlineSnapshots.identity?.identityId ?? null, value)
  if (matches) {
    transition += 1
    invalidateMemory(value.reason, false)
  }
  // The sender handles its own shared IDB transaction. Cross-app messages delete
  // only the previous owner's document, never a newer account's saved views.
  if (value.source !== 'events-admin' && value.previousIdentityId !== null) {
    await offlineSnapshots.clearIdentity(value.previousIdentityId, intentGeneration)
    publish()
  }
}
if (channel)
  channel.onmessage = (event) => {
    void receiveRevocation(event.data)
  }
if (typeof window !== 'undefined')
  window.addEventListener('storage', (event) => {
    if (event.key !== ACCOUNT_REVOCATION_KEY || !event.newValue) return
    try {
      void receiveRevocation(JSON.parse(event.newValue))
    } catch {
      /* Ignore unrelated storage changes. */
    }
  })

export async function clearPrivateSnapshots(
  reason: RevocationReason = 'session-expired',
  expectedTransition?: number,
  capturedOwner?: SnapshotOwner | null
): Promise<void> {
  if (expectedTransition !== undefined && expectedTransition !== transition) return
  const intentGeneration = currentAccountIntent()
  const lease = offlineSnapshots.lease()
  const owner =
    capturedOwner === undefined
      ? lease && offlineSnapshots.identity
        ? { ...lease, verifiedAt: offlineSnapshots.identity.verifiedAt }
        : null
      : capturedOwner
  const explicitIntent = reason === 'logout' || reason === 'account-change'
  const exactOwner =
    !!owner && offlineSnapshots.isCurrent(owner) && offlineSnapshots.identity?.verifiedAt === owner.verifiedAt
  const matches = owner
    ? explicitIntent
      ? offlineSnapshots.identity?.identityId === owner.identityId
      : exactOwner
    : offlineSnapshots.identity === null
  if (matches) {
    if (expectedTransition === undefined) transition += 1
    if (reason === 'logout' && owner && !(await noteLocalLogout(owner.identityId, intentGeneration))) return
    if (intentGeneration !== currentAccountIntent()) return
    invalidateMemory(reason, true)
  }
  // Unknown initial requests have no authority to erase a newer tab's snapshot.
  if (!owner) return
  if (explicitIntent) await offlineSnapshots.clearIdentity(owner.identityId, intentGeneration)
  else {
    const clearingTransition = transition
    const deleted = await offlineSnapshots.clearOwner(owner)
    // A newer verification may have renewed this same identity while deletion
    // was pending. A failed conditional delete with working storage also proves
    // that this owner has no authority over the current persisted document.
    if (clearingTransition !== transition || !deleted) {
      publish()
      return
    }
  }
  if (intentGeneration !== currentAccountIntent()) return
  const message: AccountRevocation = {
    kind: 'account-revoked',
    source: 'events-admin',
    reason,
    previousIdentityId: owner.identityId,
    generation: intentGeneration,
    nonce: `${Date.now()}-${Math.random().toString(36).slice(2)}`
  }
  receivedNonces.add(message.nonce)
  writeBrowserSetting(ACCOUNT_REVOCATION_KEY, JSON.stringify(message))
  channel?.postMessage(message)
  publish()
}

function scheduleExpiration(identity: OfflineIdentity) {
  clearTimeout(expirationTimer)
  if (!Number.isFinite(identity.sessionExpiresAt)) return
  expirationTimer = setTimeout(
    () => void clearPrivateSnapshots(),
    Math.max(0, snapshotExpiresAt(identity) - Date.now())
  )
}

export async function verifyOfflineIdentity(
  identity: OfflineIdentity,
  requestOwner?: SnapshotOwner | null,
  intentGeneration = currentAccountIntent()
): Promise<boolean> {
  const expected = ++transition
  if (!(await canResumeAccount(identity.identityId, intentGeneration))) return false
  if (expected !== transition) return false
  if (requestOwner !== undefined) {
    const latest = await offlineSnapshots.captureOwner(true)
    if (expected !== transition) return false
    if (
      latest &&
      latest.identityId !== identity.identityId &&
      (latest.epoch !== requestOwner?.epoch || latest.verifiedAt !== requestOwner?.verifiedAt)
    ) {
      invalidateMemory('session-expired', true)
      return false
    }
  }
  const previous = offlineSnapshots.identity?.identityId
  // On cold launch compare the persisted identity before saving a new verified one.
  if (!previous) await offlineSnapshots.restore()
  if (expected !== transition) return false
  if (offlineSnapshots.identity && offlineSnapshots.identity.identityId !== identity.identityId)
    await clearPrivateSnapshots('account-change', expected)
  if (expected !== transition) return false
  const accepted = await offlineSnapshots.verify(identity, requestOwner, intentGeneration)
  if (expected !== transition) return false
  if (!accepted) {
    invalidateMemory('session-expired', true)
    return false
  }
  if (!(await canResumeAccount(identity.identityId, intentGeneration))) return false
  if (expected !== transition) return false
  verificationNeeded = false
  publish({ mode: 'online' })
  scheduleExpiration(identity)
  return true
}

export async function restoreOfflineIdentity(): Promise<OfflineIdentity | null> {
  const expected = ++transition
  const identity =
    offlineSnapshots.identity && Number.isFinite(offlineSnapshots.identity.sessionExpiresAt)
      ? offlineSnapshots.identity
      : await offlineSnapshots.restore()
  if (expected !== transition) return null
  if (identity && wasLocallyLoggedOut(identity.identityId)) {
    await clearPrivateSnapshots('logout', expected)
    return null
  }
  if (!identity || snapshotExpiresAt(identity) <= Date.now()) {
    if (identity) await clearPrivateSnapshots('session-expired', expected)
    else invalidateMemory('session-expired', true)
    return null
  }
  verificationNeeded = true
  publish({ mode: 'offline-readonly' })
  scheduleExpiration(identity)
  return identity
}

export function markOffline(): void {
  verificationNeeded = true
  publish({
    mode:
      offlineSnapshots.identity && Number.isFinite(offlineSnapshots.identity.sessionExpiresAt)
        ? 'offline-readonly'
        : 'unverified'
  })
}

export function recordOfflineUpdate(updatedAt: number): void {
  publish({ lastUpdatedAt: updatedAt })
}

export function setSessionVerifier(verifier: () => Promise<boolean>): () => void {
  verification = verifier
  return () => {
    if (verification === verifier) verification = undefined
  }
}

export class OfflineReadUnavailable extends Error {
  constructor() {
    super('This view has no saved data. Open it while connected to make it available offline.')
    this.name = 'OfflineReadUnavailable'
  }
}

export class OfflineWriteBlocked extends Error {
  constructor() {
    super('You are in read-only mode. Reconnect and verify your session before making changes.')
    this.name = 'OfflineWriteBlocked'
  }
}

export function isOfflineReadUnavailable(error: unknown): boolean {
  return (
    error instanceof OfflineReadUnavailable || (error instanceof Error && error.cause instanceof OfflineReadUnavailable)
  )
}

export function browserIsOnline(): boolean {
  return typeof navigator === 'undefined' || navigator.onLine !== false
}

export async function requireVerifiedOnlineWrite(): Promise<IdentityLease | null> {
  if (!browserIsOnline()) throw new OfflineWriteBlocked()
  // Test/guest clients without an AuthProvider are still network-only and never persisted.
  if (typeof window === 'undefined') return null
  const lease = offlineSnapshots.lease()
  if (verificationNeeded && (!verification || !(await verification()))) throw new OfflineWriteBlocked()
  if (
    state.mode !== 'online' ||
    !offlineSnapshots.identity ||
    snapshotExpiresAt(offlineSnapshots.identity) <= Date.now()
  ) {
    throw new OfflineWriteBlocked()
  }
  if (lease && !offlineSnapshots.isCurrent(lease)) throw new OfflineWriteBlocked()
  return offlineSnapshots.lease()
}

if (typeof window !== 'undefined') {
  window.addEventListener('offline', markOffline)
  window.addEventListener('online', () => {
    verificationNeeded = true
  })
}
