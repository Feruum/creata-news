import { parsePlateJson, plateJsonToText } from '@/shared/lib/plate-utils'

export function getEventDescriptionText(description: string | null | undefined): string {
  if (!description) return ''
  try {
    const parsed: unknown = JSON.parse(description)
    if (typeof parsed === 'string') return parsed
    if (Array.isArray(parsed)) return plateJsonToText(parsePlateJson(description))
  } catch {
    // Older descriptions can be stored as plain text.
  }
  return description
}
