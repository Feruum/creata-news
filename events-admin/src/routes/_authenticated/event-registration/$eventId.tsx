import { createFileRoute } from '@tanstack/react-router'
import { RegistrationWorkspacePage } from '@/features/registration-forms'

export const Route = createFileRoute('/_authenticated/event-registration/$eventId')({
  component: EventRegistrationRoute
})

function EventRegistrationRoute() {
  const { eventId } = Route.useParams()
  return <RegistrationWorkspacePage eventId={Number(eventId)} />
}
