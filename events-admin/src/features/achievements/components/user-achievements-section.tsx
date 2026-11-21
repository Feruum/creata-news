import { useQuery } from '@connectrpc/connect-query'
import { listUserAchievements } from '@studyverse/proto'
import { Award } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { AchievementRewardDialog } from './achievement-reward-dialog'

export function UserAchievementsSection({ userId }: { userId: number }) {
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<number | null>(null)
  const query = useQuery(listUserAchievements, { userId, page, limit: 10 })
  return (
    <section className='space-y-3 rounded-lg border p-4' aria-label='User achievements'>
      <h3 className='flex items-center gap-2 font-semibold'>
        <Award size={18} aria-hidden='true' />
        Achievements
      </h3>
      {query.isPending && (
        <p role='status' className='text-sm'>
          Loading achievements…
        </p>
      )}
      {query.isError && (
        <div role='alert'>
          <p className='text-sm'>Achievements could not be loaded.</p>
          <Button variant='outline' size='sm' onClick={() => void query.refetch()}>
            Retry
          </Button>
        </div>
      )}
      {!query.isPending && !query.isError && !query.data?.achievements.length && (
        <p className='text-sm text-muted-foreground'>No achievements yet.</p>
      )}
      <ul className='space-y-2'>
        {query.data?.achievements.map(
          (award) =>
            award.achievement && (
              <li key={award.achievement.id}>
                <button
                  type='button'
                  className='w-full rounded-md border p-3 text-left hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring'
                  onClick={() => setSelected(award.achievement?.id || null)}
                >
                  <span className='block text-sm font-medium'>{award.achievement.title}</span>
                  <span className='mt-2 flex flex-wrap gap-1'>
                    {award.manual && <Badge variant='secondary'>Manual</Badge>}
                    {award.automatic && <Badge variant='secondary'>Automatic</Badge>}
                    {award.selected && <Badge>Selected badge</Badge>}
                  </span>
                </button>
              </li>
            )
        )}
      </ul>
      {(query.data?.total || 0) > 10 && (
        <div className='flex items-center justify-between'>
          <Button variant='outline' size='sm' disabled={page === 1} onClick={() => setPage((previous) => previous - 1)}>
            Previous
          </Button>
          <span className='text-xs'>Page {page}</span>
          <Button
            variant='outline'
            size='sm'
            disabled={page * 10 >= (query.data?.total || 0)}
            onClick={() => setPage((previous) => previous + 1)}
          >
            Next
          </Button>
        </div>
      )}
      {selected && (
        <AchievementRewardDialog userId={userId} achievementId={selected} onClose={() => setSelected(null)} />
      )}
    </section>
  )
}
