import { useQuery } from '@connectrpc/connect-query'
import { getUserAchievement } from '@studyverse/proto'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { AchievementCard } from './achievement-card'

export function AchievementRewardDialog({
  userId,
  achievementId,
  onClose
}: {
  userId: number
  achievementId: number
  onClose: () => void
}) {
  const query = useQuery(getUserAchievement, { userId, achievementId })
  const award = query.data?.userAchievement
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className='max-h-[92vh] overflow-y-auto sm:max-w-md' onClick={(event) => event.stopPropagation()}>
        <DialogHeader>
          <DialogTitle>{award?.achievement?.title || 'Achievement'}</DialogTitle>
          <DialogDescription>Public reward details for user #{userId}.</DialogDescription>
        </DialogHeader>
        {query.isPending && <p role='status'>Loading achievement…</p>}
        {query.isError && (
          <div role='alert' className='space-y-3'>
            <p>This reward could not be loaded. It may have been revoked.</p>
            <Button variant='outline' onClick={() => void query.refetch()}>
              Retry
            </Button>
          </div>
        )}
        {award?.achievement && (
          <>
            <AchievementCard achievement={award.achievement} />
            <div className='space-y-3 text-sm'>
              <div className='flex flex-wrap gap-2'>
                {award.manual && <Badge variant='secondary'>Manual award</Badge>}
                {award.automatic && <Badge variant='secondary'>Automatic award</Badge>}
                {award.selected && <Badge>Selected name badge</Badge>}
              </div>
              <p className='whitespace-pre-wrap break-words'>{award.achievement.description}</p>
              {award.basis && (
                <p className='break-words'>
                  <span className='font-medium'>Awarded for: </span>
                  {award.basis}
                </p>
              )}
              <p className='text-muted-foreground'>Awarded {new Date(award.awardedAt).toLocaleDateString()}</p>
            </div>
          </>
        )}
        {!query.isPending && !query.isError && !award && (
          <p className='text-sm text-muted-foreground'>This user no longer holds this achievement.</p>
        )}
      </DialogContent>
    </Dialog>
  )
}
