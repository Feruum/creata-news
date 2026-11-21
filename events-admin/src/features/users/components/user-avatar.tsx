import { Blobatar } from '@blobatar/react'
import { useState } from 'react'
import { cn } from '@/shared/ui/lib/utils'

interface UserAvatarProps {
  userId: number
  username: string
  name?: string
  avatarUrl?: string | null
  telegramUsername?: string | null
  className?: string
}

export function UserAvatar({ userId, username, name, avatarUrl, telegramUsername, className }: UserAvatarProps) {
  const [failedImageUrl, setFailedImageUrl] = useState<string | null>(null)

  const imageUrl = avatarUrl || (telegramUsername ? `https://t.me/i/userpic/320/${telegramUsername}.jpg` : null)
  const displayName = name || username
  const avatarClassName = cn('h-10 w-10 shrink-0 rounded-full', className)

  const handleImageLoad = (e: React.SyntheticEvent<HTMLImageElement>) => {
    const img = e.currentTarget
    if (img.naturalWidth === 1 && img.naturalHeight === 1) {
      setFailedImageUrl(imageUrl)
    }
  }

  if (!imageUrl || failedImageUrl === imageUrl) {
    return (
      <div className={cn('flex items-center justify-center overflow-hidden bg-muted', avatarClassName)}>
        <Blobatar
          name={`user-${userId}`}
          title={displayName}
          animate='always'
          background={false}
          className='size-full'
        />
      </div>
    )
  }

  return (
    <img
      key={imageUrl}
      alt={displayName}
      className={cn(avatarClassName, 'object-cover')}
      onError={() => setFailedImageUrl(imageUrl)}
      onLoad={handleImageLoad}
      src={imageUrl}
    />
  )
}
