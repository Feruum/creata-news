import assert from 'node:assert/strict'
import { test } from 'node:test'
import { matchesCommand } from './command-matching'

test('local navigation remains searchable independently of remote content results', () => {
  for (const query of ['', 'Calendar', 'calendar', ' CAL ', '/calendar', 'G C']) {
    assert.equal(matchesCommand(query, 'Calendar', '/calendar', 'G C'), true)
  }
  assert.equal(matchesCommand('unrelated event', 'Calendar', '/calendar', 'G C'), false)
})
