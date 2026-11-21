import type { IEvent } from './interfaces'

export async function persistCalendarEvent(
  original: IEvent,
  updated: IEvent,
  apply: (event: IEvent) => void,
  persist: (input: { id: number; startTime: string; endTime: string }) => Promise<unknown>
) {
  apply(updated)
  try {
    await persist({ id: updated.id, startTime: updated.startDate, endTime: updated.endDate })
  } catch (error) {
    apply(original)
    throw error
  }
}
