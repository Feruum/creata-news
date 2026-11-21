import { createClient } from '@connectrpc/connect'
import { useTransport } from '@connectrpc/connect-query'
import { RegistrationFormsService } from '@studyverse/proto'
import { RegistrationWorkspace } from '@studyverse/registration-forms'
import { useMemo } from 'react'
import { useAuth } from '@/shared/lib/auth'
import { buttonVariants } from '@/shared/ui/components/button'
import { useRegistrationSessionExpiry } from '../hooks/use-registration-session-expiry'
import '@studyverse/registration-forms/styles.css'

export function RegistrationWorkspacePage({ eventId }: { eventId: number }) {
  const auth = useAuth()
  const transport = useTransport()
  const client = useMemo(() => createClient(RegistrationFormsService, transport), [transport])
  const expired = useRegistrationSessionExpiry(auth.session?.expires_at, auth.checkSession)

  if (!Number.isSafeInteger(eventId) || eventId <= 0) {
    return <p role='alert'>Мероприятие не найдено.</p>
  }
  if (expired) return <p role='alert'>Сессия завершена. Войдите в аккаунт снова.</p>
  if (auth.isLoading) return <p role='status'>Проверяем вход…</p>
  if (!auth.isAuthenticated || !auth.session?.identity?.id) {
    return <p role='alert'>Для работы с формой и ответами войдите в аккаунт и подключитесь к интернету.</p>
  }

  return (
    <div className='rf-workspace-page space-y-4'>
      <a href={`${import.meta.env.BASE_URL}events?event=${eventId}`} className={buttonVariants({ variant: 'outline' })}>
        К мероприятию
      </a>
      <RegistrationWorkspace key={`${auth.session.identity.id}:${eventId}`} eventId={eventId} client={client} />
    </div>
  )
}
