import assert from 'node:assert/strict'
import { test } from 'node:test'
import { Code, ConnectError } from '@connectrpc/connect'
import { handleServerErrors } from './form-utils'

test('owned forms receive one root error even when server field details are empty', () => {
  const errors: Array<{ field: string; message?: string }> = []
  const error = new ConnectError('internal booking details', Code.FailedPrecondition)
  error.metadata.set('error-details', JSON.stringify({ errors: [] }))
  handleServerErrors<{ title: string }>(
    {
      setError: (field, detail) => {
        errors.push({ field, message: detail.message })
      }
    },
    error,
    { showToast: false, context: 'booking' }
  )
  assert.deepEqual(errors, [
    {
      field: 'root',
      message: 'Room approval is required for this room and time. Check the booking status and try again.'
    }
  ])
})

test('server field errors map to the visible form field without a second root error', () => {
  const errors: Array<{ field: string; message?: string }> = []
  const error = new ConnectError('invalid room', Code.InvalidArgument)
  error.metadata.set('error-details', JSON.stringify({ errors: [{ field: 'room_id', message: 'Choose a room' }] }))
  handleServerErrors<{ roomId: string }>(
    {
      setError: (field, detail) => {
        errors.push({ field, message: detail.message })
      }
    },
    error,
    { showToast: false, fieldMapping: { room_id: 'roomId' } }
  )
  assert.deepEqual(errors, [{ field: 'roomId', message: 'Choose a room' }])
})
