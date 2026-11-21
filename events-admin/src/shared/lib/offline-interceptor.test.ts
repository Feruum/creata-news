import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { Code, ConnectError, createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { UsersService } from '@studyverse/proto'
import { offlineInterceptor } from './offline-interceptor'
import { clearPrivateSnapshots, getOfflineState, offlineSnapshots, verifyOfflineIdentity } from './offline-runtime'

const identity = (id: string) => ({
  identityId: id,
  displayName: id,
  email: `${id}@example.test`,
  verifiedAt: Date.now(),
  sessionExpiresAt: Date.now() + 60_000
})

afterEach(async () => {
  await clearPrivateSnapshots()
})

test('a late RPC 401 from account A cannot clear the verified identity B', async () => {
  let release: ((response: Response) => void) | undefined
  let requested: (() => void) | undefined
  const started = new Promise<void>((resolve) => {
    requested = resolve
  })
  const response = new Promise<Response>((resolve) => {
    release = resolve
  })
  const client = createClient(
    UsersService,
    createConnectTransport({
      baseUrl: 'https://api.example.test',
      useBinaryFormat: false,
      interceptors: [offlineInterceptor],
      fetch: async () => {
        requested?.()
        return response
      }
    })
  )
  await verifyOfflineIdentity(identity('a'))
  const oldRequest = client.getMe({})
  await started
  await verifyOfflineIdentity(identity('b'))
  release?.(Response.json({ code: 'unauthenticated', message: 'Old request' }, { status: 401 }))
  await assert.rejects(oldRequest, (error: unknown) => error instanceof ConnectError && error.code === Code.Canceled)
  assert.equal(getOfflineState().identity?.identityId, 'b')
  assert.equal(getOfflineState().mode, 'online')
})

test('a superseded verification cannot invalidate the account verified while its save was pending', async () => {
  let release: ((accepted: boolean) => void) | undefined
  let requested: (() => void) | undefined
  const started = new Promise<void>((resolve) => {
    requested = resolve
  })
  const pending = new Promise<boolean>((resolve) => {
    release = resolve
  })
  await verifyOfflineIdentity(identity('a'))
  const originalVerify = offlineSnapshots.verify
  offlineSnapshots.verify = async function (candidate, owner) {
    if (candidate.identityId === 'a') {
      requested?.()
      return pending
    }
    return originalVerify.call(this, candidate, owner)
  }
  try {
    const oldVerification = verifyOfflineIdentity(identity('a'))
    await started
    await verifyOfflineIdentity(identity('b'))
    release?.(false)
    assert.equal(await oldVerification, false)
    assert.equal(getOfflineState().identity?.identityId, 'b')
    assert.equal(getOfflineState().mode, 'online')
  } finally {
    offlineSnapshots.verify = originalVerify
  }
})

test('an RPC response from A is rejected if the account changes while persistence is pending', async () => {
  let release: ((stored: boolean) => void) | undefined
  let requested: (() => void) | undefined
  const started = new Promise<void>((resolve) => {
    requested = resolve
  })
  const pending = new Promise<boolean>((resolve) => {
    release = resolve
  })
  const originalSave = offlineSnapshots.save
  offlineSnapshots.save = async () => {
    requested?.()
    return pending
  }
  try {
    const client = createClient(
      UsersService,
      createConnectTransport({
        baseUrl: 'https://api.example.test',
        useBinaryFormat: false,
        interceptors: [offlineInterceptor],
        fetch: async () => Response.json({ user: { id: 1 } })
      })
    )
    await verifyOfflineIdentity(identity('a'))
    const oldRequest = client.getMe({})
    await started
    await verifyOfflineIdentity(identity('b'))
    release?.(true)
    await assert.rejects(oldRequest, (error: unknown) => error instanceof ConnectError && error.code === Code.Canceled)
    assert.equal(getOfflineState().identity?.identityId, 'b')
  } finally {
    offlineSnapshots.save = originalSave
  }
})
