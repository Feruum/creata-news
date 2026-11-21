import assert from 'node:assert/strict'
import { test } from 'node:test'
import { getActorName } from './actor-name'

test('distinguishes initiators with missing profiles from system actions', () => {
  assert.equal(getActorName({ display_name: 'Ada', username: 'ada-user' }, 7), 'Ada')
  assert.equal(getActorName({ display_name: ' ', username: 'ada-user' }, 7), 'ada-user')
  assert.equal(getActorName({}, 7), 'User #7')
  assert.equal(getActorName({ display_name: 42 }, 7), 'User #7')
  assert.equal(getActorName({}), 'System')
  assert.equal(getActorName({}, 0), 'System')
})
