export const CALENDAR_PAGE_SIZE = 200

export async function loadCalendarEvents<T>(
  loadPage: (input: { page: number; limit: number }) => Promise<{ events: T[]; total: number }>
): Promise<{ events: T[] }> {
  const events: T[] = []
  for (let page = 1; ; page += 1) {
    const result = await loadPage({ page, limit: CALENDAR_PAGE_SIZE })
    events.push(...result.events)
    if (result.events.length === 0 || events.length >= result.total) return { events }
  }
}
