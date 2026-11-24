import { ApiError } from '@creata/contracts'

const pendingReads = new Map<string, Promise<boolean>>()

export function acknowledgeStoryUpdates(userId: string, write: () => Promise<unknown>) {
  const result = Promise.resolve(pendingReads.get(userId)).then(async () => {
    for (let attempt = 0; attempt < 2; attempt++) {
      try {
        await write()
        return true
      } catch (error) {
        if (error instanceof ApiError && error.status < 500) return false
      }
    }
    return false
  })
  pendingReads.set(userId, result)
  void result.then(() => {
    if (pendingReads.get(userId) === result) pendingReads.delete(userId)
  })
  return result
}

export async function waitForStoryReads(userId: string) {
  await pendingReads.get(userId)
}
