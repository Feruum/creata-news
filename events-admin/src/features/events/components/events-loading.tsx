import { Skeleton } from '@/shared/ui/components/skeleton'

const skeletonRowKeys = Array.from({ length: 10 }, (_, index) => `skeleton-row-${index}`)
const skeletonPageKeys = Array.from({ length: 4 }, (_, index) => `skeleton-page-${index}`)

export function EventsLoading() {
  return (
    <div className='space-y-4'>
      <div>
        <Skeleton className='h-7 w-24' />
        <Skeleton className='h-4 w-48 mt-1' />
      </div>

      <div className='flex items-center justify-between gap-4'>
        <div className='flex items-center gap-2'>
          <Skeleton className='h-9 w-64' />
          <Skeleton className='h-9 w-20' />
        </div>
        <div className='flex items-center gap-2'>
          <Skeleton className='h-9 w-20' />
          <Skeleton className='h-9 w-20' />
          <Skeleton className='h-9 w-24' />
        </div>
      </div>

      <div className='rounded-lg border'>
        <div className='border-b px-4 py-3'>
          <Skeleton className='h-4 w-full max-w-2xl' />
        </div>
        {skeletonRowKeys.map((key) => (
          <div key={key} className='border-b last:border-0 px-4 py-3'>
            <Skeleton className='h-5 w-full' />
          </div>
        ))}
      </div>

      <div className='flex items-center justify-between'>
        <Skeleton className='h-9 w-28' />
        <Skeleton className='h-4 w-32' />
        <div className='flex gap-1'>
          {skeletonPageKeys.map((key) => (
            <Skeleton key={key} className='h-9 w-9' />
          ))}
        </div>
      </div>
    </div>
  )
}
