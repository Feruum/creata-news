import { sqlClient } from '../db'

export async function withSingletonLease(lockId: number, task: () => Promise<void>): Promise<boolean> {
  const connection = await sqlClient.reserve()
  try {
    const [result] = await connection<{ acquired: boolean }[]>`
      SELECT pg_try_advisory_lock(${lockId}) AS acquired
    `
    if (!result?.acquired) return false
    try {
      await task()
      return true
    } finally {
      await connection`SELECT pg_advisory_unlock(${lockId})`
    }
  } finally {
    connection.release()
  }
}
