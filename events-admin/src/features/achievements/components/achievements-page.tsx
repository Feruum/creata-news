import { useQuery } from '@connectrpc/connect-query'
import { type Achievement, AchievementStatus, listAchievements } from '@studyverse/proto'
import { Award, Plus, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { useDebounce } from '@/shared/hooks/use-debounce'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import { useAchievementClient, useRefreshAchievements } from '../api'
import { AchievementAwardsDialog } from './achievement-awards-dialog'
import { AchievementEditorDialog } from './achievement-editor-dialog'

export function AchievementsPage() {
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search.trim(), 250)
  const [status, setStatus] = useState('all')
  const [page, setPage] = useState(1)
  const [editor, setEditor] = useState<Achievement | 'new' | null>(null)
  const [awards, setAwards] = useState<Achievement | null>(null)
  const [pending, setPending] = useState<number | null>(null)
  const [feedback, setFeedback] = useState('')
  const client = useAchievementClient()
  const refresh = useRefreshAchievements()
  const query = useQuery(listAchievements, {
    page,
    limit: 20,
    search: debouncedSearch,
    status: status === 'all' ? undefined : Number(status)
  })

  async function changeStatus(achievement: Achievement, nextStatus: AchievementStatus) {
    setPending(achievement.id)
    try {
      await client.saveAchievement({ achievement: { ...achievement, status: nextStatus } })
      await refresh()
      toast.success(
        nextStatus === AchievementStatus.ARCHIVED
          ? 'Achievement archived. Existing awards are preserved.'
          : 'Achievement activated'
      )
    } catch (cause) {
      toast.error(cause instanceof Error ? cause.message : 'Could not update status')
    } finally {
      setPending(null)
    }
  }

  async function recalculate(achievement: Achievement) {
    setPending(achievement.id)
    setFeedback('Recalculating automatic eligibility…')
    try {
      const response = await client.recalculateAchievements({ achievementId: achievement.id })
      setFeedback(
        response.queued
          ? 'Recalculation queued. Award history will reflect changes when processing completes.'
          : 'Recalculation complete. Award history has been refreshed.'
      )
      await refresh()
    } catch (cause) {
      setFeedback(cause instanceof Error ? cause.message : 'Recalculation failed')
    } finally {
      setPending(null)
    }
  }

  return (
    <div className='space-y-6'>
      <div className='flex flex-wrap items-start justify-between gap-4'>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>Achievements</h1>
          <p className='mt-1 text-sm text-muted-foreground'>Create rewards, manage criteria, and audit awards.</p>
        </div>
        <Button onClick={() => setEditor('new')}>
          <Plus aria-hidden='true' />
          Create achievement
        </Button>
      </div>
      <div className='flex flex-wrap items-end gap-3'>
        <div className='min-w-56 flex-1 space-y-2'>
          <Label htmlFor='achievements-search'>Search achievements</Label>
          <Input
            id='achievements-search'
            placeholder='Search titles and descriptions…'
            value={search}
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
            }}
          />
        </div>
        <div className='space-y-2'>
          <Label htmlFor='achievements-status'>Status</Label>
          <select
            id='achievements-status'
            className='h-9 rounded-md border bg-background px-3 text-sm'
            value={status}
            onChange={(event) => {
              setStatus(event.target.value)
              setPage(1)
            }}
          >
            <option value='all'>All statuses</option>
            <option value={AchievementStatus.DRAFT}>Draft</option>
            <option value={AchievementStatus.ACTIVE}>Active</option>
            <option value={AchievementStatus.ARCHIVED}>Archived</option>
          </select>
        </div>
      </div>
      {feedback && (
        <p role='status' className='rounded-md border p-3 text-sm'>
          {feedback}
        </p>
      )}
      {query.isPending && <p role='status'>Loading achievements…</p>}
      {query.isError && (
        <div role='alert' className='space-y-3'>
          <p>Achievements could not be loaded.</p>
          <Button variant='outline' onClick={() => void query.refetch()}>
            Retry
          </Button>
        </div>
      )}
      {query.data && (
        <>
          <div className='overflow-x-auto rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Achievement</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Criterion</TableHead>
                  <TableHead className='text-right'>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.data.achievements.map((achievement) => (
                  <TableRow key={achievement.id}>
                    <TableCell>
                      <div className='flex items-center gap-3'>
                        <Award size={20} aria-hidden='true' style={{ color: achievement.accentColor }} />
                        <div>
                          <p className='font-medium'>{achievement.title}</p>
                          <p className='max-w-72 truncate text-xs text-muted-foreground'>
                            {achievement.description || `#${achievement.id}`}
                          </p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={achievement.status === AchievementStatus.ACTIVE ? 'default' : 'secondary'}>
                        {
                          {
                            [AchievementStatus.DRAFT]: 'Draft',
                            [AchievementStatus.ACTIVE]: 'Active',
                            [AchievementStatus.ARCHIVED]: 'Archived',
                            [AchievementStatus.UNSPECIFIED]: 'Unknown'
                          }[achievement.status]
                        }
                      </Badge>
                    </TableCell>
                    <TableCell className='text-sm'>
                      {achievement.rule
                        ? `${achievement.rule.threshold} ${['', 'registrations', 'attendances', 'SOC GPA attendances', 'IROS attendances'][achievement.rule.kind]}`
                        : 'Manual only'}
                    </TableCell>
                    <TableCell>
                      <div className='flex flex-wrap justify-end gap-2'>
                        <Button size='sm' variant='outline' onClick={() => setEditor(achievement)}>
                          Edit
                        </Button>
                        <Button size='sm' variant='outline' onClick={() => setAwards(achievement)}>
                          Awards & history
                        </Button>
                        {achievement.status !== AchievementStatus.ACTIVE && (
                          <Button
                            size='sm'
                            variant='outline'
                            disabled={pending !== null}
                            onClick={() => void changeStatus(achievement, AchievementStatus.ACTIVE)}
                          >
                            Activate
                          </Button>
                        )}
                        {achievement.status !== AchievementStatus.ARCHIVED && (
                          <Button
                            size='sm'
                            variant='outline'
                            disabled={pending !== null}
                            onClick={() => void changeStatus(achievement, AchievementStatus.ARCHIVED)}
                          >
                            Archive
                          </Button>
                        )}
                        {achievement.rule && achievement.status === AchievementStatus.ACTIVE && (
                          <Button
                            size='sm'
                            variant='outline'
                            disabled={pending !== null}
                            onClick={() => void recalculate(achievement)}
                          >
                            <RefreshCw size={14} aria-hidden='true' />
                            Recalculate
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {!query.data.achievements.length && (
                  <TableRow>
                    <TableCell colSpan={4} className='h-32 text-center text-muted-foreground'>
                      {search || status !== 'all'
                        ? 'No achievements match these filters.'
                        : 'No achievements yet. Create the first reward to get started.'}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
          <div className='flex items-center justify-between gap-3 text-sm'>
            <Button variant='outline' disabled={page === 1} onClick={() => setPage((previous) => previous - 1)}>
              Previous
            </Button>
            <span>
              {query.data.total} achievements · Page {page}
            </span>
            <Button
              variant='outline'
              disabled={page * 20 >= query.data.total}
              onClick={() => setPage((previous) => previous + 1)}
            >
              Next
            </Button>
          </div>
        </>
      )}
      {editor && (
        <AchievementEditorDialog achievement={editor === 'new' ? undefined : editor} onClose={() => setEditor(null)} />
      )}
      {awards && <AchievementAwardsDialog achievement={awards} onClose={() => setAwards(null)} />}
    </div>
  )
}
