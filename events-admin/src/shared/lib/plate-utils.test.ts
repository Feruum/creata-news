import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parsePlateJson, plateJsonToText } from './plate-utils'

test('event editor and details retain plain descriptions from map and legacy storage', () => {
  assert.equal(plateJsonToText(parsePlateJson('"Map event description"')), 'Map event description')
  assert.equal(plateJsonToText(parsePlateJson('Legacy plain description')), 'Legacy plain description')
  assert.equal(plateJsonToText(parsePlateJson('""')), '')
})
