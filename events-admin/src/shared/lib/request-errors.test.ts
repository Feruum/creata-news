import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Code, ConnectError } from '@connectrpc/connect'
import { OfflineWriteBlocked } from './offline-runtime'
import { getRequestErrorMessage } from './request-errors'

test('RPC failures display guidance without raw protocol or server details', () => {
  assert.equal(
    getRequestErrorMessage(new ConnectError('database booking_id=3', Code.FailedPrecondition)),
    'This action is not available yet. Refresh and try again.'
  )
  assert.equal(
    getRequestErrorMessage(new ConnectError('SQL connection failed', Code.Unavailable)),
    'Could not connect to the server. Check your connection and try again.'
  )
  assert.equal(
    getRequestErrorMessage(new ConnectError('internal policy 42', Code.PermissionDenied)),
    'You do not have permission to perform this action.'
  )
  assert.equal(
    getRequestErrorMessage(new TypeError('Failed to fetch')),
    'Could not connect to the server. Check your connection and try again.'
  )
  assert.equal(
    getRequestErrorMessage(new ConnectError('sql details', Code.FailedPrecondition), 'booking'),
    'Room approval is required for this room and time. Check the booking status and try again.'
  )
})

test('offline write errors retain the read-only recovery guidance', () => {
  const blocked = new OfflineWriteBlocked()
  assert.equal(getRequestErrorMessage(blocked), blocked.message)
  assert.equal(getRequestErrorMessage(ConnectError.from(blocked)), blocked.message)
})
