import { createFileRoute } from '@tanstack/react-router'
import { AchievementsPage } from '@/features/achievements'
import { useRequireAdmin } from '@/shared/lib/use-current-user'
import { Button } from '@/shared/ui/components/button'

export const Route = createFileRoute('/_authenticated/achievements')({ component: AchievementsRoute })

function AchievementsRoute() {
  const { hasAccess, isLoading, error, refetch } = useRequireAdmin()
  if (isLoading) return <p role='status'>Checking administrator access…</p>
  if (error)
    return (
      <div role='alert' className='space-y-3'>
        <p>Administrator access could not be checked.</p>
        <Button variant='outline' onClick={refetch}>
          Retry
        </Button>
      </div>
    )
  if (!hasAccess)
    return (
      <div className='space-y-3'>
        <h1 className='text-2xl font-semibold'>Administrator access required</h1>
        <p className='text-sm text-muted-foreground'>Only platform administrators can manage achievements.</p>
      </div>
    )
  return <AchievementsPage />
}
