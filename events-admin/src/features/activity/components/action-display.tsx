import {
  Ban,
  BookOpenCheck,
  Building2,
  CalendarPlus,
  DoorOpen,
  Globe,
  Pencil,
  Plus,
  Trash,
  UserMinus,
  UserPlus,
  Users,
  type LucideIcon
} from 'lucide-react'

import { Badge } from '@/shared/ui/components/badge'

export type ActionConfig = {
  label: string
  icon: LucideIcon
  color: 'green' | 'blue' | 'red' | 'yellow' | 'purple' | 'gray'
}

export const ACTION_MAP: Record<string, ActionConfig> = {
  'event.created': { label: 'Created event', icon: Plus, color: 'green' },
  'event.updated': { label: 'Updated event', icon: Pencil, color: 'blue' },
  'event.published': { label: 'Published event', icon: Globe, color: 'green' },
  'event.cancelled': { label: 'Cancelled event', icon: Ban, color: 'red' },
  'event.deleted': { label: 'Deleted event', icon: Trash, color: 'red' },
  'participant.registered': { label: 'Participant registered', icon: UserPlus, color: 'green' },
  'participant.unregistered': { label: 'Participant unregistered', icon: UserMinus, color: 'yellow' },
  'org.created': { label: 'Created organization', icon: Building2, color: 'green' },
  'org.updated': { label: 'Updated organization', icon: Pencil, color: 'blue' },
  'org.deleted': { label: 'Deleted organization', icon: Trash, color: 'red' },
  'member.invited': { label: 'Invited member', icon: UserPlus, color: 'purple' },
  'member.removed': { label: 'Removed member', icon: UserMinus, color: 'red' },
  'member.role_changed': { label: 'Changed member role', icon: Users, color: 'blue' },
  'room.booking_requested': { label: 'Requested booking', icon: CalendarPlus, color: 'yellow' },
  'room.booking_approved': { label: 'Approved booking', icon: BookOpenCheck, color: 'green' },
  'room.booking_rejected': { label: 'Rejected booking', icon: DoorOpen, color: 'red' }
}

const COLOR_CLASSES: Record<string, string> = {
  green: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  blue: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  red: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  yellow: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400',
  purple: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
  gray: 'bg-muted text-muted-foreground'
}

export function ActionDisplay({ action }: { action: string }) {
  const config = ACTION_MAP[action] ?? {
    label: action,
    icon: CalendarPlus,
    color: 'gray' as const
  }

  const Icon = config.icon

  return (
    <Badge variant='outline' className={`gap-1.5 font-normal ${COLOR_CLASSES[config.color]}`}>
      <Icon className='size-3.5' />
      {config.label}
    </Badge>
  )
}

export function getActionLabel(action: string): string {
  return ACTION_MAP[action]?.label ?? action
}

export const ACTION_GROUPS = {
  Events: ['event.created', 'event.updated', 'event.published', 'event.cancelled', 'event.deleted'],
  Participants: ['participant.registered', 'participant.unregistered'],
  Organizations: ['org.created', 'org.updated', 'org.deleted'],
  Members: ['member.invited', 'member.removed', 'member.role_changed'],
  Bookings: ['room.booking_requested', 'room.booking_approved', 'room.booking_rejected']
} as const

export const ALL_ACTIONS = Object.values(ACTION_GROUPS).flat()
