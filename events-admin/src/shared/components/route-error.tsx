import { ErrorComponent, type ErrorComponentProps } from '@tanstack/react-router'
import { isOfflineReadUnavailable } from '@/shared/lib/offline-runtime'
import { Button } from '@/shared/ui/components/button'

export function RouteError(props: ErrorComponentProps) {
  if (!isOfflineReadUnavailable(props.error)) return <ErrorComponent {...props} />
  return (
    <div role='alert' className='rounded-lg border p-6'>
      <h2 className='text-lg font-semibold'>This view is unavailable offline</h2>
      <p className='mt-2 text-sm text-muted-foreground'>
        This view has no saved data. Open it while connected to make it available offline.
      </p>
      <Button className='mt-4' variant='outline' onClick={props.reset}>
        Try again
      </Button>
    </div>
  )
}
