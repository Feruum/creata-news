import { PlatformRole, type User } from '@studyverse/proto'
import { format, parseISO } from 'date-fns'
import { CalendarDays, Mail, Shield, XIcon } from 'lucide-react'
import { Suspense, useState } from 'react'
import { UserAchievementsSection } from '@/features/achievements'
import { UserMembershipTimeline } from '@/features/organizations'
import { useUser } from '@/features/users'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/shared/ui/components/dialog'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { Telegram } from '@/shared/ui/components/svgs/telegram'
import { cn } from '@/shared/ui/lib/utils'
import { UserAvatar } from './user-avatar'

function getRoleBadgeClass(role: PlatformRole): string {
  switch (role) {
    case PlatformRole.ADMIN:
      return 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400'
    case PlatformRole.STAFF:
      return 'bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-400'
    case PlatformRole.USER:
      return 'bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
  }
}

function getRoleLabel(role: PlatformRole): string {
  switch (role) {
    case PlatformRole.ADMIN:
      return 'Admin'
    case PlatformRole.STAFF:
      return 'Staff'
    case PlatformRole.USER:
      return 'User'
    default:
      return 'Unknown'
  }
}

interface UserAvatarLargeProps {
  user: User
}

function UserAvatarLarge({ user }: UserAvatarLargeProps) {
  const displayName =
    user.firstName || user.lastName ? `${user.firstName || ''} ${user.lastName || ''}`.trim() : user.username

  return (
    <UserAvatar
      userId={user.id}
      username={user.username}
      name={displayName}
      avatarUrl={user.avatarUrl}
      telegramUsername={user.telegramUsername}
      className='h-24 w-24'
    />
  )
}

interface UserProfileContentProps {
  user: User
  onClose: () => void
}

function UserProfileContent({ user, onClose }: UserProfileContentProps) {
  const createdAt = user.createdAt ? format(parseISO(user.createdAt), 'MMMM d, yyyy') : 'Unknown'
  const displayName =
    user.firstName || user.lastName ? `${user.firstName || ''} ${user.lastName || ''}`.trim() : user.username

  return (
    <>
      <div className='flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4'>
        <div className='min-w-0 flex-1'>
          <div className='flex items-center gap-2 mb-1'>
            <Badge variant='secondary' className={cn('text-xs', getRoleBadgeClass(user.platformRole))}>
              {getRoleLabel(user.platformRole)}
            </Badge>
            <span className='text-xs text-muted-foreground'>ID #{user.id}</span>
          </div>
          <div className='flex items-center gap-2'>
            <DialogTitle className='text-lg font-semibold leading-tight truncate'>{displayName}</DialogTitle>
            <AchievementNameBadge badge={user.selectedBadge} userId={user.id} />
          </div>
          <DialogDescription className='sr-only'>User profile for {displayName}</DialogDescription>
        </div>
        <Button variant='ghost' size='icon' aria-label='Close' className='h-8 w-8 shrink-0' onClick={onClose}>
          <XIcon className='h-4 w-4' aria-hidden='true' />
        </Button>
      </div>

      <div className='flex-1 overflow-y-auto p-6'>
        <div className='flex gap-6 mb-6'>
          <UserAvatarLarge user={user} />
          <div className='flex-1 min-w-0 space-y-3'>
            <div>
              <p className='text-sm text-muted-foreground'>Username</p>
              <p className='font-medium'>@{user.username}</p>
            </div>
            {(user.firstName || user.lastName) && (
              <div>
                <p className='text-sm text-muted-foreground'>Full Name</p>
                <p className='font-medium'>{displayName}</p>
              </div>
            )}
          </div>
        </div>

        <div className='space-y-4'>
          <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
            <div className='flex items-start gap-4'>
              <div className='flex h-12 w-12 items-center justify-center rounded-lg bg-primary/10 text-primary'>
                <Mail className='h-6 w-6' aria-hidden='true' />
              </div>
              <div className='flex-1 min-w-0'>
                <p className='text-sm font-semibold'>Email</p>
                <p className='text-sm text-muted-foreground truncate'>{user.email}</p>
              </div>
            </div>
          </div>

          <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
            <div className='flex items-start gap-4'>
              <div className='flex h-12 w-12 items-center justify-center rounded-lg bg-purple-100 text-purple-600 dark:bg-purple-950 dark:text-purple-400'>
                <Shield className='h-6 w-6' aria-hidden='true' />
              </div>
              <div className='flex-1 min-w-0'>
                <p className='text-sm font-semibold'>Platform Role</p>
                <p className='text-sm text-muted-foreground'>{getRoleLabel(user.platformRole)}</p>
              </div>
            </div>
          </div>

          <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
            <div className='flex items-start gap-4'>
              <div className='flex h-12 w-12 items-center justify-center rounded-lg bg-blue-100 text-blue-600 dark:bg-blue-950 dark:text-blue-400'>
                <CalendarDays className='h-6 w-6' aria-hidden='true' />
              </div>
              <div className='flex-1 min-w-0'>
                <p className='text-sm font-semibold'>Member Since</p>
                <p className='text-sm text-muted-foreground'>{createdAt}</p>
              </div>
            </div>
          </div>

          {user.telegramUsername && (
            <div className='rounded-lg border bg-gradient-to-br from-muted/50 to-muted/30 p-4'>
              <div className='flex items-start gap-4'>
                <div className='flex h-12 w-12 items-center justify-center rounded-lg bg-sky-100 text-sky-600 dark:bg-sky-950 dark:text-sky-400'>
                  <Telegram className='h-6 w-6' aria-hidden='true' />
                </div>
                <div className='flex-1 min-w-0'>
                  <p className='text-sm font-semibold'>Telegram</p>
                  <a
                    href={`https://t.me/${user.telegramUsername}`}
                    target='_blank'
                    rel='noopener noreferrer'
                    className='text-sm text-primary hover:underline'
                  >
                    @{user.telegramUsername}
                  </a>
                  {user.telegramLinkedAt && (
                    <p className='text-xs text-muted-foreground mt-1'>
                      Linked {format(parseISO(user.telegramLinkedAt), 'MMM d, yyyy')}
                    </p>
                  )}
                </div>
              </div>
            </div>
          )}
          <UserMembershipTimeline userId={user.id} />
          <UserAchievementsSection userId={user.id} />
        </div>
      </div>
    </>
  )
}

function UserProfileSkeleton() {
  return (
    <>
      <div className='flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4'>
        <div className='flex-1 space-y-2'>
          <div className='flex items-center gap-2'>
            <Skeleton className='h-5 w-12 rounded-full' />
            <Skeleton className='h-4 w-12' />
          </div>
          <Skeleton className='h-6 w-40' />
        </div>
        <Skeleton className='h-8 w-8 rounded-md' />
      </div>

      <div className='flex-1 overflow-y-auto p-6'>
        <div className='flex gap-6 mb-6'>
          <Skeleton className='h-24 w-24 shrink-0 rounded-full' />
          <div className='flex-1 space-y-3'>
            <div className='space-y-1'>
              <Skeleton className='h-4 w-16' />
              <Skeleton className='h-5 w-24' />
            </div>
            <div className='space-y-1'>
              <Skeleton className='h-4 w-16' />
              <Skeleton className='h-5 w-32' />
            </div>
          </div>
        </div>
        <div className='space-y-4'>
          <Skeleton className='h-20 w-full rounded-lg' />
          <Skeleton className='h-20 w-full rounded-lg' />
          <Skeleton className='h-20 w-full rounded-lg' />
        </div>
      </div>
    </>
  )
}

interface UserProfileModalProps {
  userId: number
  children: React.ReactNode
}

export function UserProfileModal({ userId, children }: UserProfileModalProps) {
  const [open, setOpen] = useState(false)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <button type='button' onClick={() => setOpen(true)} className='cursor-pointer text-left'>
        {children}
      </button>
      <DialogContent
        className='flex max-h-[90vh] w-[min(500px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <Suspense fallback={<UserProfileSkeleton />}>
          <UserProfileModalContent userId={userId} onClose={() => setOpen(false)} />
        </Suspense>
      </DialogContent>
    </Dialog>
  )
}

function UserProfileModalContent({ userId, onClose }: { userId: number; onClose: () => void }) {
  const { data } = useUser(userId)
  const user = data?.user

  if (!user) {
    return (
      <div className='flex h-64 items-center justify-center'>
        <p className='text-sm text-muted-foreground'>User not found</p>
      </div>
    )
  }

  return <UserProfileContent user={user} onClose={onClose} />
}

interface UserProfileModalControlledProps {
  userId: number | null
  onClose: () => void
}

export function UserProfileModalControlled({ userId, onClose }: UserProfileModalControlledProps) {
  if (!userId) return null

  return (
    <Dialog open={!!userId} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        className='flex max-h-[90vh] w-[min(500px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <Suspense fallback={<UserProfileSkeleton />}>
          <UserProfileModalContent userId={userId} onClose={onClose} />
        </Suspense>
      </DialogContent>
    </Dialog>
  )
}
