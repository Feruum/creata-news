import { EventFormat } from '@studyverse/proto'
import { Badge } from '@/shared/ui/components/badge'

interface FormatBadgeProps {
  format: EventFormat
}

export function FormatBadge({ format }: FormatBadgeProps) {
  switch (format) {
    case EventFormat.ONLINE:
      return (
        <Badge
          variant='secondary'
          className='gap-1.5 bg-green-50 text-green-700 hover:bg-green-100 border-green-200 shrink-0 font-medium'
        >
          <span className='h-1.5 w-1.5 rounded-full bg-green-600' />
          Online
        </Badge>
      )
    case EventFormat.HYBRID:
      return (
        <Badge
          variant='secondary'
          className='gap-1.5 bg-purple-50 text-purple-700 hover:bg-purple-100 border-purple-200 shrink-0 font-medium'
        >
          <span className='h-1.5 w-1.5 rounded-full bg-purple-600' />
          Hybrid
        </Badge>
      )
    case EventFormat.OFFLINE:
      return (
        <Badge
          variant='secondary'
          className='gap-1.5 bg-gray-100 text-gray-700 hover:bg-gray-200 border-gray-200 shrink-0 font-medium'
        >
          <span className='h-1.5 w-1.5 rounded-full bg-gray-500' />
          Offline
        </Badge>
      )
    default:
      return null
  }
}
