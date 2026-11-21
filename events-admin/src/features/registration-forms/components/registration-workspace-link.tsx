import { ClipboardList } from 'lucide-react'
import { buttonVariants } from '@/shared/ui/components/button'

interface RegistrationWorkspaceLinkProps {
  eventId: number
  canEdit: boolean
  canManageRegistrations: boolean
}

export function RegistrationWorkspaceLink({
  eventId,
  canEdit,
  canManageRegistrations
}: RegistrationWorkspaceLinkProps) {
  if (!canEdit && !canManageRegistrations) return null

  return (
    <a
      href={`${import.meta.env.BASE_URL ?? '/'}event-registration/${eventId}`}
      className={buttonVariants({ variant: 'ghost', size: 'icon-sm' })}
      title='Форма и ответы'
      aria-label='Форма и ответы'
    >
      <ClipboardList className='h-4 w-4' aria-hidden='true' />
    </a>
  )
}
