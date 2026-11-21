import assert from 'node:assert/strict'
import { test } from 'node:test'
import {
  safeProductAttributes,
  sanitizeErrorStack,
  sanitizeSpanAttributes,
  sanitizeTelemetryUrl
} from './telemetry-events'

test('product events accept only bounded attributes and never input text or identities', () => {
  assert.deepEqual(
    safeProductAttributes({
      surface: 'map',
      category: 'room',
      outcome: 'success',
      floor: '2',
      source: 'someone@example.com',
      query: 'private text',
      userId: 42,
      cookie: 'secret'
    }),
    { surface: 'map', category: 'room', outcome: 'success', floor: '2' }
  )
})

test('telemetry URLs remove credentials, queries, fragments and entity identifiers', () => {
  assert.equal(
    sanitizeTelemetryUrl('https://user:secret@map.webspace.cat/administration/users/42?token=secret#private'),
    'https://map.webspace.cat/administration/users/:id'
  )
  assert.equal(
    sanitizeTelemetryUrl('https://map.webspace.cat/api/threads/thread-private-name/messages?page=token'),
    'https://map.webspace.cat/api/threads/:id/messages'
  )
})

test('span sanitization removes SQL and free-form exception content without losing RPC status', () => {
  assert.deepEqual(
    sanitizeSpanAttributes({
      'http.url': 'https://map.webspace.cat/api/ems?email=private',
      'http.status_code': 500,
      'rpc.method': 'ListEvents',
      'db.statement': "SELECT * FROM users WHERE email = 'private'",
      'http.request.header.cookie': 'private',
      'exception.message': 'password=secret',
      'enduser.id': 17
    }),
    {
      'http.url': 'https://map.webspace.cat/api/ems',
      'http.status_code': 500,
      'rpc.method': 'ListEvents'
    }
  )
})

test('browser error stacks omit message text and scrub URL parameters', () => {
  assert.equal(
    sanitizeErrorStack('Error: secret password\n    at run (https://map.webspace.cat/app.js?token=secret:2:3)'),
    '    at run (https://map.webspace.cat/app.js)'
  )
})

test('multiline error input including fake stack frames cannot enter browser logs', () => {
  const message = 'first line\nprivate text\n    at private-input (https://example.test/private)'
  const stack = `Error: ${message}\n    at run (https://map.webspace.cat/app.js?token=secret:2:3)`
  assert.equal(sanitizeErrorStack(stack, message), '    at run (https://map.webspace.cat/app.js)')
})
