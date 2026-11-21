import type { EventLocation } from '@studyverse/proto'
import { LocationType } from '@studyverse/proto'

export function formatStructuredLocation(loc?: EventLocation): string {
  if (!loc) return ''

  switch (loc.type) {
    case LocationType.CAMPUS_ROOM:
      if (loc.roomName) {
        return loc.floor != null ? `${loc.roomName}, Floor ${loc.floor}` : loc.roomName
      }
      return ''

    case LocationType.EXTERNAL:
      return loc.externalAddress ?? ''

    case LocationType.ONLINE:
      return loc.onlinePlatform ?? (loc.onlineUrl ? 'Online' : '')

    case LocationType.HYBRID: {
      const parts: string[] = []
      if (loc.roomName) {
        parts.push(loc.roomName)
      } else if (loc.externalAddress) {
        parts.push(loc.externalAddress)
      }
      parts.push(loc.onlinePlatform ?? 'Online')
      return parts.join(' + ')
    }

    default:
      return ''
  }
}
