import { Building2, CalendarDays, DoorOpen, type LucideIcon, UserRound, Users } from 'lucide-react'
import { useState } from 'react'

export function parseSnapshot(json: string): Record<string, unknown> {
  try {
    return JSON.parse(json)
  } catch {
    return {}
  }
}

export function formatTimeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes} min ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days === 1) return 'yesterday'
  return `${days}d ago`
}

export function getActionVerb(action: string): string {
  const verbs: Record<string, string> = {
    'event.created': 'created',
    'event.updated': 'updated',
    'event.published': 'published',
    'event.cancelled': 'cancelled',
    'event.deleted': 'deleted',
    'participant.registered': 'registered for',
    'participant.unregistered': 'unregistered from',
    'org.created': 'created',
    'org.updated': 'updated',
    'org.deleted': 'deleted',
    'member.invited': 'invited',
    'member.removed': 'removed',
    'member.role_changed': 'changed role of',
    'room.booking_requested': 'requested booking',
    'room.booking_approved': 'approved booking',
    'room.booking_rejected': 'rejected booking'
  }
  return verbs[action] ?? action.split('.').pop() ?? action
}

export const ENTITY_TYPE_ICONS: Record<string, LucideIcon> = {
  event: CalendarDays,
  organization: Building2,
  participant: UserRound,
  member: Users,
  room: DoorOpen
}

export const BADGE_COLOR_CLASSES: Record<string, string> = {
  green: 'text-emerald-600 dark:text-emerald-400',
  blue: 'text-blue-600 dark:text-blue-400',
  red: 'text-red-600 dark:text-red-400',
  yellow: 'text-amber-600 dark:text-amber-400',
  purple: 'text-purple-600 dark:text-purple-400',
  gray: 'text-muted-foreground'
}

export function ActorAvatar({ name, telegramUsername }: { name: string; telegramUsername?: string }) {
  const [imageError, setImageError] = useState(false)

  const avatarUrl = telegramUsername
    ? `https://t.me/i/userpic/320/${telegramUsername}.jpg`
    : `https://api.dicebear.com/9.x/glass/svg?seed=${encodeURIComponent(name)}`

  const isTelegram = !!telegramUsername

  const handleImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    if (isTelegram) {
      const img = e.currentTarget
      if (img.naturalWidth === 1 && img.naturalHeight === 1) {
        setImageError(true)
      }
    }
  }

  const telegramAvatarFailed = isTelegram && imageError
  if (telegramAvatarFailed) {
    return (
      <img
        alt={name}
        className='size-7 shrink-0 rounded-full border'
        src={`https://api.dicebear.com/9.x/glass/svg?seed=${encodeURIComponent(name)}`}
      />
    )
  }

  return (
    <img
      alt={name}
      className='size-7 shrink-0 rounded-full border'
      onError={() => setImageError(true)}
      onLoad={handleImageLoad}
      src={avatarUrl}
    />
  )
}
