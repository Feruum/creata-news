export function getActorName(snapshot: Record<string, unknown>, actorId?: number): string {
  for (const key of ['display_name', 'username']) {
    const value = snapshot[key]
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return actorId && actorId > 0 ? `User #${actorId}` : 'System'
}
