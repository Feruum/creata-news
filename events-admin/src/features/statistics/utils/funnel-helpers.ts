export function formatLevelName(level: string): string {
  return level.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
}

export function calculateDropOffRate(prevPercentage: number, currentPercentage: number): number {
  return currentPercentage - prevPercentage
}

export function formatFunnelLabel(count: number, percentage: number): string {
  return `${count.toLocaleString()} users (${percentage.toFixed(1)}%)`
}

export function formatDropOff(rate: number): string {
  const sign = rate >= 0 ? '+' : ''
  return `${sign}${rate.toFixed(1)}%`
}

export function getInsightPlaceholder(stageName: string): string {
  const insights: Record<string, string> = {
    'Active Users':
      'These are all registered users in the system. Consider engagement campaigns to move them to the next stage.',
    'Registered For Events':
      'Users who have registered for at least one event. Focus on event reminders and quality content.',
    'Attended Events': 'Users who attended at least one event. Encourage feedback and showcase upcoming events.',
    'Repeat Attendees': 'Your most engaged users. Consider loyalty programs and ambassador opportunities.'
  }
  return insights[stageName] || `Engagement insights for ${stageName} will appear here.`
}

export function getDropOffColor(rate: number): string {
  if (rate >= 0) return 'text-green-600 dark:text-green-400'
  if (rate > -20) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

export function getDropOffBgColor(rate: number): string {
  if (rate >= 0) return 'bg-green-500/10'
  if (rate > -20) return 'bg-amber-500/10'
  return 'bg-red-500/10'
}
