import { Link } from '@tanstack/react-router'

export function StatisticsAccess() {
  return (
    <div className='space-y-3'>
      <h1 className='text-2xl font-semibold'>Доступ к статистике ограничен</h1>
      <p className='text-sm text-muted-foreground'>Статистика доступна администраторам и сотрудникам EMS.</p>
      <Link to='/events' className='text-primary underline'>
        Перейти к событиям
      </Link>
    </div>
  )
}
