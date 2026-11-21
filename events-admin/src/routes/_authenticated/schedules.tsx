import { createFileRoute, Link } from '@tanstack/react-router'
import { ScheduleModerationPage } from '@/features/schedules'
import { useRequireAdmin } from '@/shared/lib/use-current-user'
import { Button } from '@/shared/ui/components/button'

export const Route = createFileRoute('/_authenticated/schedules')({ component: SchedulesRoute })

function SchedulesRoute() {
  const { hasAccess, isLoading, error, refetch } = useRequireAdmin()

  if (isLoading)
    return (
      <p role='status' className='text-sm text-muted-foreground'>
        Checking admin access...
      </p>
    )
  if (error) {
    return (
      <div role='alert' className='space-y-3'>
        <p>Admin access could not be checked. Please try again.</p>
        <Button variant='outline' onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    )
  }
  if (!hasAccess) {
    return (
      <div className='space-y-3'>
        <h1 className='text-2xl font-semibold'>Admin access required</h1>
        <p className='text-sm text-muted-foreground'>Only administrators can review and publish group schedules.</p>
        <Button variant='outline' render={<Link to='/calendar' />} nativeButton={false}>
          Open calendar
        </Button>
      </div>
    )
  }
  return <ScheduleModerationPage />
}
