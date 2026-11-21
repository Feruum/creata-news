import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getLoginReturnUrl } from './auth-utils'

test('login returns to the requested route under the deployed admin base path', () => {
  const origin = 'https://map.webspace.cat'
  assert.equal(getLoginReturnUrl('/organizations?tab=members', '/administration/', origin), `${origin}/administration/organizations?tab=members`)
  assert.equal(getLoginReturnUrl('/administration/events', '/administration/', origin), `${origin}/administration/events`)
  assert.equal(getLoginReturnUrl(undefined, '/administration/', origin), `${origin}/administration/`)
  assert.equal(getLoginReturnUrl('https://other.example/events', '/administration/', origin), `${origin}/administration/`)
  assert.equal(getLoginReturnUrl('/events', '/', origin), `${origin}/events`)
})
