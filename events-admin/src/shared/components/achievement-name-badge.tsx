import type { NameBadge } from '@studyverse/proto'
import { Award } from 'lucide-react'
import { useState } from 'react'
import { AchievementRewardDialog } from '@/features/achievements'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/components/tooltip'

export function AchievementNameBadge({
  badge,
  userId,
  inline = false
}: {
  badge?: NameBadge
  userId: number
  inline?: boolean
}) {
  const [open, setOpen] = useState(false)
  const [failedImage, setFailedImage] = useState<string | null>(null)
  const Control = inline ? 'span' : 'button'
  if (!badge) return null
  return (
    // biome-ignore lint/a11y/noStaticElementInteractions: Isolates events from the badge and its portaled dialog; the child is the accessible control.
    <span
      className='inline-flex align-middle'
      onPointerDown={(event) => event.stopPropagation()}
      onMouseDown={(event) => event.stopPropagation()}
      onClick={(event) => event.stopPropagation()}
      onKeyDown={(event) => event.stopPropagation()}
      onKeyUp={(event) => event.stopPropagation()}
    >
      <Tooltip>
        <TooltipTrigger
          render={
            <Control
              type={inline ? undefined : 'button'}
              role='button'
              tabIndex={0}
              className='inline-flex size-6 shrink-0 items-center justify-center rounded-full border bg-background align-middle focus-visible:outline-2 focus-visible:outline-ring'
              style={{ color: badge.accentColor, borderColor: badge.accentColor }}
              aria-label={`View ${badge.title} achievement for user ${userId}`}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  setOpen(true)
                }
              }}
              onClick={() => setOpen(true)}
            />
          }
        >
          {badge.iconUrl && failedImage !== badge.iconUrl ? (
            <img
              src={badge.iconUrl}
              alt=''
              className='size-4 rounded-full object-contain'
              onError={() => setFailedImage(badge.iconUrl)}
            />
          ) : (
            <Award size={15} aria-hidden='true' />
          )}
        </TooltipTrigger>
        <TooltipContent>{badge.title}</TooltipContent>
      </Tooltip>
      {open && (
        <AchievementRewardDialog userId={userId} achievementId={badge.achievementId} onClose={() => setOpen(false)} />
      )}
    </span>
  )
}
