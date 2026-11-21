export function isLessonReminderDue(start: string, now = Date.now()): boolean {
  const remaining = Date.parse(start) - now
  return remaining > 60_000 && remaining <= 15 * 60_000
}

export function campusDate(now = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Almaty',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).format(now)
}
