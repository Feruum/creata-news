import { createFileRoute } from '@tanstack/react-router'
import { Construction } from 'lucide-react'

export const Route = createFileRoute('/_authenticated/reports')({
  component: Reports
})

function Reports() {
  return (
    <div className='flex flex-1 flex-col items-center justify-center'>
      <Construction className='h-12 w-12 text-muted-foreground mb-4' />
      <h1 className='text-xl font-semibold mb-2'>Reports Coming Soon</h1>
      <p className='text-muted-foreground text-sm text-center'>This feature is under development.</p>
    </div>
  )
}
