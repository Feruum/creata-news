import { expect,test } from 'bun:test'
import { countdownParts } from './countdown'
test('opportunity countdown crosses day boundaries and never displays negative or invalid values',()=>{
  const now=Date.parse('2026-11-14T09:00:00Z')
  expect(countdownParts('2026-11-15T11:03:04Z',now)).toEqual([1,2,3,4])
  expect(countdownParts('2026-11-15T09:00:00Z',now+1)).toEqual([0,23,59,59])
  expect(countdownParts('2026-11-13T09:00:00Z',now)).toEqual([0,0,0,0])
  expect(countdownParts('invalid',now)).toEqual([0,0,0,0])
})
