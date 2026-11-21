import { useQuery } from '@connectrpc/connect-query'
import {
  type Achievement,
  globalSearch,
  listAchievementHistory,
  listUserAchievements,
  listUsers,
  SearchResultType
} from '@studyverse/proto'
import { useState } from 'react'
import { toast } from 'sonner'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { useDebounce } from '@/shared/hooks/use-debounce'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Textarea } from '@/shared/ui/components/textarea'
import { useAchievementClient, useRefreshAchievements } from '../api'
import { selectedRecipientIds, validateAwardAction } from '../validation'

export function AchievementAwardsDialog({ achievement, onClose }: { achievement: Achievement; onClose: () => void }) {
  const client = useAchievementClient()
  const refresh = useRefreshAchievements()
  const [page, setPage] = useState(1)
  const [historyPage, setHistoryPage] = useState(1)
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search.trim(), 250)
  const [selection, setSelection] = useState<Record<string, boolean>>({})
  const [names, setNames] = useState<Record<number, string>>({})
  const [reason, setReason] = useState('')
  const [operation, setOperation] = useState<'grant' | 'revoke'>('grant')
  const [pending, setPending] = useState(false)
  const [error, setError] = useState('')
  const users = useQuery(listUsers, { page, limit: 20 }, { enabled: !debouncedSearch })
  const results = useQuery(
    globalSearch,
    { query: debouncedSearch, limit: 50, types: [SearchResultType.USER] },
    { enabled: Boolean(debouncedSearch) }
  )
  const history = useQuery(listAchievementHistory, { achievementId: achievement.id, page: historyPage, limit: 20 })
  const recipients = selectedRecipientIds(selection)
  const selectedAward = useQuery(
    listUserAchievements,
    { userId: recipients[0] || 0, achievementId: achievement.id, limit: 1, page: 1 },
    { enabled: recipients.length === 1, retry: false }
  )
  const options = debouncedSearch
    ? results.data?.results
        .filter((result) => result.type === SearchResultType.USER)
        .map((result) => ({
          id: result.id,
          name: result.title,
          detail: result.description,
          selectedBadge: result.selectedBadge
        })) || []
    : users.data?.users.map((user) => ({
        id: user.id,
        name: `${user.firstName || ''} ${user.lastName || ''}`.trim() || user.username,
        detail: user.email,
        selectedBadge: user.selectedBadge
      })) || []

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const validationError = validateAwardAction(recipients, reason)
    if (validationError) {
      setError(validationError)
      return
    }
    setPending(true)
    setError('')
    try {
      const response = await client[operation === 'grant' ? 'grantAchievement' : 'revokeAchievement']({
        achievementId: achievement.id,
        userIds: recipients,
        reason: reason.trim()
      })
      await refresh()
      setSelection({})
      setReason('')
      toast.success(
        `${operation === 'grant' ? 'Granted or restored' : 'Revoked'} for ${response.affected} user${response.affected === 1 ? '' : 's'}`
      )
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'The award update failed')
    } finally {
      setPending(false)
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose()
      }}
    >
      <DialogContent className='max-h-[92vh] overflow-y-auto sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle>Manage awards · {achievement.title}</DialogTitle>
          <DialogDescription>
            Grant, restore, or revoke this achievement for existing users. Every change is recorded.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className='space-y-4'>
          <fieldset disabled={pending} className='space-y-4'>
            <div className='space-y-2'>
              <Label htmlFor='award-search'>Find recipients</Label>
              <Input
                id='award-search'
                value={search}
                placeholder='Search users by name or email…'
                onChange={(event) => setSearch(event.target.value)}
              />
            </div>
            {(debouncedSearch ? results.isPending : users.isPending) && <p role='status'>Loading users…</p>}
            {(debouncedSearch ? results.isError : users.isError) && (
              <div role='alert'>
                <p>Users could not be loaded.</p>
                <Button
                  type='button'
                  variant='outline'
                  onClick={() => void (debouncedSearch ? results.refetch() : users.refetch())}
                >
                  Retry
                </Button>
              </div>
            )}
            <div className='max-h-60 overflow-y-auto rounded-lg border divide-y'>
              {options.map((user) => (
                <div key={user.id} className='flex items-center gap-3 px-3 py-2 text-sm'>
                  <label className='flex min-w-0 flex-1 cursor-pointer items-center gap-3'>
                    <input
                      type='checkbox'
                      checked={Boolean(selection[user.id])}
                      onChange={(event) => {
                        setSelection((previous) => ({ ...previous, [user.id]: event.target.checked }))
                        setNames((previous) => ({ ...previous, [user.id]: user.name }))
                      }}
                    />
                    <span className='min-w-0 flex-1'>
                      <span className='block truncate font-medium'>{user.name}</span>
                      <span className='block truncate text-xs text-muted-foreground'>
                        {user.detail} · #{user.id}
                      </span>
                    </span>
                  </label>
                  <AchievementNameBadge badge={user.selectedBadge} userId={user.id} />
                </div>
              ))}
              {!options.length && !(debouncedSearch ? results.isPending : users.isPending) && (
                <p className='p-3 text-sm text-muted-foreground'>No users found.</p>
              )}
            </div>
            {!debouncedSearch && (
              <div className='flex items-center justify-between gap-3 text-sm'>
                <Button
                  type='button'
                  variant='outline'
                  disabled={page === 1 || users.isPending}
                  onClick={() => setPage((previous) => previous - 1)}
                >
                  Previous users
                </Button>
                <span>
                  Page {page} · {users.data?.total ?? 0} users
                </span>
                <Button
                  type='button'
                  variant='outline'
                  disabled={page * 20 >= (users.data?.total || 0) || users.isPending}
                  onClick={() => setPage((previous) => previous + 1)}
                >
                  Next users
                </Button>
              </div>
            )}
            {debouncedSearch && (
              <p className='text-xs text-muted-foreground'>
                Up to 50 matching users. Refine your search to narrow the results.
              </p>
            )}
            <div className='space-y-2'>
              <p className='text-sm font-medium'>{recipients.length} selected</p>
              <div className='flex flex-wrap gap-2'>
                {recipients.map((id) => (
                  <Button
                    key={id}
                    variant='secondary'
                    size='sm'
                    type='button'
                    aria-label={`Remove ${names[id] || `user ${id}`} from recipients`}
                    onClick={() => setSelection((previous) => ({ ...previous, [id]: false }))}
                  >
                    {names[id] || `#${id}`} ×
                  </Button>
                ))}
              </div>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='award-operation'>Action</Label>
              <select
                id='award-operation'
                className='h-9 w-full rounded-md border bg-background px-3 text-sm'
                value={operation}
                onChange={(event) => setOperation(event.target.value === 'revoke' ? 'revoke' : 'grant')}
              >
                <option value='grant'>Grant or restore achievement</option>
                <option value='revoke'>Revoke achievement</option>
              </select>
            </div>
            {recipients.length === 1 && (
              <p role='status' className='text-xs text-muted-foreground'>
                {selectedAward.isPending
                  ? 'Checking current award…'
                  : selectedAward.isError
                    ? 'Current award status could not be checked.'
                    : selectedAward.data?.achievements[0]
                      ? `Current award: ${[selectedAward.data.achievements[0].manual && 'manual', selectedAward.data.achievements[0].automatic && 'automatic'].filter(Boolean).join(' and ')}${selectedAward.data.achievements[0].selected ? ' · selected name badge' : ''}`
                      : 'This user does not currently hold this achievement.'}
              </p>
            )}
            <p className='text-sm text-muted-foreground'>
              {operation === 'revoke'
                ? 'Revocation removes the award and its selected badge, and blocks automatic re-awards until an administrator restores it by granting again.'
                : 'Granting restores revoked awards and allows automatic evaluation again.'}
            </p>
            <div className='space-y-2'>
              <Label htmlFor='award-reason'>Reason (required for audit history)</Label>
              <Textarea
                id='award-reason'
                required
                maxLength={2000}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
              />
            </div>
            {error && (
              <p role='alert' className='text-sm text-destructive'>
                {error}
              </p>
            )}
            <Button
              type='submit'
              variant={operation === 'revoke' ? 'destructive' : 'default'}
              disabled={pending || !recipients.length || !reason.trim()}
            >
              {pending
                ? 'Updating…'
                : `${operation === 'grant' ? 'Grant / restore' : 'Revoke'} for ${recipients.length} selected`}
            </Button>
          </fieldset>
        </form>
        <section className='space-y-3 border-t pt-5' aria-label='Award history'>
          <h3 className='font-semibold'>Award history</h3>
          {history.isPending && <p role='status'>Loading history…</p>}
          {history.isError && (
            <div role='alert'>
              <p>History could not be loaded.</p>
              <Button variant='outline' onClick={() => void history.refetch()}>
                Retry
              </Button>
            </div>
          )}
          {!history.isPending && !history.isError && !history.data?.entries.length && (
            <p className='text-sm text-muted-foreground'>No award changes yet.</p>
          )}
          <ol className='space-y-3'>
            {history.data?.entries.map((entry) => (
              <li key={entry.id.toString()} className='rounded-md border p-3 text-sm'>
                <p className='font-medium'>
                  {entry.action} · User #{entry.userId}
                </p>
                <p className='break-words'>{entry.reason || 'Automatic criterion evaluation'}</p>
                <p className='mt-1 text-xs text-muted-foreground'>
                  {entry.actorId ? `Administrator #${entry.actorId}` : 'Automatic'} ·{' '}
                  {new Date(entry.createdAt).toLocaleString()}
                </p>
              </li>
            ))}
          </ol>
          <div className='flex items-center justify-between gap-3 text-sm'>
            <Button
              variant='outline'
              disabled={historyPage === 1 || history.isPending}
              onClick={() => setHistoryPage((previous) => previous - 1)}
            >
              Previous history
            </Button>
            <span>Page {historyPage}</span>
            <Button
              variant='outline'
              disabled={historyPage * 20 >= (history.data?.total || 0) || history.isPending}
              onClick={() => setHistoryPage((previous) => previous + 1)}
            >
              Next history
            </Button>
          </div>
        </section>
      </DialogContent>
    </Dialog>
  )
}
