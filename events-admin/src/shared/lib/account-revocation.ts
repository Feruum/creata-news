import { accountIntents } from './account-intent'
import { readBrowserSetting, writeBrowserSetting } from './browser-storage'

export const ACCOUNT_CHANNEL = 'studyverse-accounts'
export const ACCOUNT_REVOCATION_KEY = 'studyverse-private-revocation'
const logoutKey = 'studyverse-logout-pending'
export type RevocationReason = 'logout' | 'account-change' | 'session-expired'
export interface AccountRevocation {
  kind: 'account-revoked'
  source: 'map' | 'events-admin'
  reason: RevocationReason
  previousIdentityId: string | null
  nonce: string
  generation?: string
}

export interface AccountLoginReset {
  kind: 'account-login-reset'
  source: 'map' | 'events-admin'
  generation: string
  nonce: string
}

export function parseAccountLoginReset(value: unknown): AccountLoginReset | null {
  if (!value || typeof value !== 'object') return null
  const object = value as Record<string, unknown>
  return object.kind === 'account-login-reset' &&
    (object.source === 'map' || object.source === 'events-admin') &&
    typeof object.generation === 'string' &&
    /^\d+:/.test(object.generation) &&
    typeof object.nonce === 'string'
    ? (object as unknown as AccountLoginReset)
    : null
}

export const captureAccountIntent = () => accountIntents.restore()
export const currentAccountIntent = () => accountIntents.currentGeneration()
export const canResumeAccount = (identityId: string, checkpoint: string) =>
  accountIntents.canResume(identityId, checkpoint, readBrowserSetting(logoutKey))
export const applyAccountLoginReset = (message: AccountLoginReset) => accountIntents.acceptReset(message.generation)

export function isAccountRevocation(value: unknown): value is AccountRevocation {
  if (!value || typeof value !== 'object') return false
  return (
    'kind' in value &&
    value.kind === 'account-revoked' &&
    'source' in value &&
    (value.source === 'map' || value.source === 'events-admin') &&
    'reason' in value &&
    (value.reason === 'logout' || value.reason === 'account-change' || value.reason === 'session-expired') &&
    'previousIdentityId' in value &&
    (typeof value.previousIdentityId === 'string' || value.previousIdentityId === null) &&
    'nonce' in value &&
    typeof value.nonce === 'string' &&
    (!('generation' in value) || value.generation === undefined || typeof value.generation === 'string')
  )
}

export function shouldRevokeIdentity(identityId: string | null, message: AccountRevocation): boolean {
  return identityId !== null && identityId === message.previousIdentityId
}

const initialLogout = readBrowserSetting(logoutKey)
if (initialLogout) accountIntents.pendingIdentities.add(initialLogout)
export function noteLocalLogout(identityId: string | null, generation = currentAccountIntent()) {
  const intent = identityId ?? 'true'
  writeBrowserSetting(logoutKey, intent)
  return accountIntents.remember(intent, generation)
}

export async function resetLocalLogout(broadcast = true) {
  writeBrowserSetting(logoutKey, null)
  const generation = await accountIntents.reset()
  if (broadcast && typeof window !== 'undefined' && typeof BroadcastChannel !== 'undefined') {
    const channel = new BroadcastChannel(ACCOUNT_CHANNEL)
    channel.postMessage({ kind: 'account-login-reset', source: 'events-admin', generation, nonce: crypto.randomUUID() })
    channel.close()
  }
  return generation
}

export function wasLocallyLoggedOut(identityId: string): boolean {
  // A failed quota write may leave storage empty or holding an older account's
  // marker. Only an explicit reset or storage event can clear local intent.
  const storedLogout = readBrowserSetting(logoutKey)
  return accountIntents.isBlocked(identityId, storedLogout)
}

if (typeof window !== 'undefined')
  window.addEventListener('storage', (event) => {
    if (event.key !== logoutKey) return
    if (event.newValue === null) void accountIntents.acceptStorageReset()
    else accountIntents.pendingIdentities.add(event.newValue)
  })
