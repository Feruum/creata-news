import { CalendarCheck, Loader2, RefreshCw } from 'lucide-react'
import { useRef, useState } from 'react'
import { Button } from '@/shared/ui/components/button'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/components/tabs'
import { scheduleErrorMessage } from '../api/client'
import { useScheduleSubmissions } from '../api/queries'
import type { SubmissionFilters } from '../types'
import { ScheduleReviewDialog } from './schedule-review-dialog'
import { ScheduleSubmissionsTable } from './schedule-submissions-table'

export function ScheduleModerationPage() {
  const [status, setStatus] = useState<SubmissionFilters['status']>('pending')
  const [groupInput, setGroupInput] = useState('')
  const [group, setGroup] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const returnFocusRef = useRef<HTMLButtonElement | null>(null)
  const query = useScheduleSubmissions({ status, ...(group ? { group } : {}) })
  const submissions = query.data?.pages.flatMap((page) => page.submissions) ?? []

  return (
    <div className='min-w-0 space-y-6'>
      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Schedule moderation</h1>
        <p className='mt-1 text-sm text-muted-foreground'>Review MyDU imports before publishing group schedules.</p>
      </div>
      <Tabs
        value={status}
        onValueChange={(value) => {
          setStatus(value === 'all' ? 'all' : 'pending')
          setSelectedId(null)
        }}
      >
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <TabsList aria-label='Schedule submission status'>
            <TabsTrigger value='pending'>Pending</TabsTrigger>
            <TabsTrigger value='all'>History</TabsTrigger>
          </TabsList>
          <Button variant='outline' size='sm' disabled={query.isFetching} onClick={() => void query.refetch()}>
            <RefreshCw aria-hidden='true' className='size-4' />
            Refresh
          </Button>
        </div>
        <form
          className='my-4 flex flex-wrap items-end gap-2'
          onSubmit={(event) => {
            event.preventDefault()
            setGroup(groupInput.trim())
            setSelectedId(null)
          }}
        >
          <div className='min-w-0 flex-1 space-y-1.5 sm:max-w-xs'>
            <Label htmlFor='schedule-group-filter'>Group</Label>
            <Input
              id='schedule-group-filter'
              value={groupInput}
              onChange={(event) => setGroupInput(event.target.value)}
              placeholder='For example, SE-2401'
            />
          </div>
          <Button variant='outline' type='submit'>
            Filter
          </Button>
          {group && (
            <Button
              variant='ghost'
              type='button'
              onClick={() => {
                setGroupInput('')
                setGroup('')
              }}
            >
              Clear filter
            </Button>
          )}
        </form>
        {(['pending', 'all'] as const).map((tab) => (
          <TabsContent key={tab} value={tab} className='min-w-0 space-y-4'>
            {tab === 'all' && (
              <p className='text-sm text-muted-foreground'>All submissions, including pending and previous reviews.</p>
            )}
            {query.isPending ? (
              <p role='status' className='flex items-center gap-2 py-8 text-sm text-muted-foreground'>
                <Loader2 aria-hidden='true' className='size-4 animate-spin' />
                Loading schedule submissions...
              </p>
            ) : query.isError ? (
              <div role='alert' className='space-y-3 rounded-lg border p-4 text-sm'>
                <p>{scheduleErrorMessage(query.error)}</p>
                <Button variant='outline' disabled={query.isFetching} onClick={() => void query.refetch()}>
                  Retry
                </Button>
              </div>
            ) : submissions.length ? (
              <>
                <ScheduleSubmissionsTable
                  submissions={submissions}
                  onSelect={(id, trigger) => {
                    returnFocusRef.current = trigger
                    setSelectedId(id)
                  }}
                />
                {query.hasNextPage && (
                  <Button
                    variant='outline'
                    disabled={query.isFetchingNextPage}
                    onClick={() => void query.fetchNextPage()}
                  >
                    {query.isFetchingNextPage && <Loader2 aria-hidden='true' className='size-4 animate-spin' />}
                    Load more submissions
                  </Button>
                )}
              </>
            ) : (
              <div className='flex flex-col items-center rounded-lg border px-4 py-12 text-center'>
                <CalendarCheck aria-hidden='true' className='mb-3 size-10 text-muted-foreground' />
                <h2 className='font-medium'>
                  {tab === 'pending' ? 'No pending schedules' : 'No schedule submissions'}
                </h2>
                <p className='mt-1 text-sm text-muted-foreground'>
                  {group
                    ? 'No submissions match this group. Try another group or clear the filter.'
                    : tab === 'pending'
                      ? 'New MyDU imports will appear here for review.'
                      : 'Submitted MyDU imports and review decisions will appear here.'}
                </p>
              </div>
            )}
          </TabsContent>
        ))}
      </Tabs>
      {selectedId && (
        <ScheduleReviewDialog
          key={selectedId}
          id={selectedId}
          onClose={() => setSelectedId(null)}
          returnFocusRef={returnFocusRef}
        />
      )}
    </div>
  )
}
