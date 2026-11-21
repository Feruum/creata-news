import { Skeleton } from '@/shared/ui/components/skeleton'
import { DialogDescription, DialogTitle } from '@/shared/ui/components/dialog'

export function EventDetailsSkeleton() {
  return (
    <>
      <DialogTitle className='sr-only'>Event details</DialogTitle>
      <DialogDescription className='sr-only'>Loading event details</DialogDescription>
      <div className='flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4'>
        <div className='flex-1 space-y-2'>
          <Skeleton className='h-4 w-20' />
          <Skeleton className='h-6 w-3/4' />
        </div>
        <div className='flex gap-1'>
          <Skeleton className='h-8 w-8 rounded-md' />
          <Skeleton className='h-8 w-8 rounded-md' />
          <Skeleton className='h-8 w-8 rounded-md' />
          <Skeleton className='h-8 w-8 rounded-md' />
        </div>
      </div>

      <div className='flex-1 overflow-y-auto p-6'>
        <div className='flex gap-6 mb-6'>
          <Skeleton className='h-48 w-48 shrink-0 rounded-xl' />
          <div className='flex-1 space-y-4'>
            <Skeleton className='h-20 w-full rounded-lg' />
            <Skeleton className='h-20 w-full rounded-lg' />
          </div>
        </div>
        <div className='grid gap-4 md:grid-cols-[200px_1fr]'>
          <div className='space-y-4'>
            <Skeleton className='h-24 w-full rounded-lg' />
            <Skeleton className='h-20 w-full rounded-lg' />
          </div>
          <Skeleton className='h-44 w-full rounded-lg' />
        </div>
      </div>
    </>
  )
}
