import { create } from '@bufbuild/protobuf'
import { Code, ConnectError, createClient, createRouterTransport } from '@connectrpc/connect'
import { TransportProvider } from '@connectrpc/connect-query'
import {
  ApprovalMode,
  EventAttendanceService,
  EventRegistrationsService,
  EventResponseSchema,
  EventStatus,
  EventsService,
  FormFieldType,
  type RegistrationForm,
  RegistrationFormSchema,
  RegistrationFormsService,
  RegistrationReviewAction,
  RegistrationSource,
  RegistrationStatus,
  sundayopenModels,
  sundayopenServices,
  UsersService
} from '@studyverse/proto'
import { RegistrationWorkspace } from '@studyverse/registration-forms'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createRoot } from 'react-dom/client'
import { EventDetailsModalControlled } from '@/features/events'
import { RegistrationWorkspacePage } from '@/features/registration-forms'
import { AuthProvider, useAuth } from '@/shared/lib/auth'
import '@/index.css'

const params = new URLSearchParams(location.search)
const seeded = params.has('seeded') || params.has('design')
const calls: string[] = []
let form = create(RegistrationFormSchema, {
  eventId: 42,
  revision: 1,
  approvalMode: ApprovalMode.AUTOMATIC,
  filesAvailable: true,
  fields: seeded
    ? [
        { id: 'motivation', label: 'Почему вы хотите участвовать?', type: FormFieldType.LONG_TEXT, required: true },
        {
          id: 'experience',
          label: 'Какой у вас опыт?',
          type: FormFieldType.SINGLE_CHOICE,
          required: true,
          options: [
            { id: 'beginner', label: 'Только начинаю' },
            { id: 'some', label: 'Есть небольшой опыт' },
            { id: 'regular', label: 'Занимаюсь регулярно' }
          ]
        },
        { id: 'portfolio', label: 'Портфолио', type: FormFieldType.FILE }
      ]
    : []
})
let published: RegistrationForm | undefined
let response = create(EventResponseSchema, {
  registration: { source: RegistrationSource.SUNDAY_OPEN, id: 99 },
  responseId: 'response-99',
  contact: seeded
    ? { fullName: 'Алия Садыкова', email: 'aliya@example.test', telegramUsername: 'aliya_sadykova' }
    : { fullName: 'Sunday participant', email: 'sunday@example.test' },
  status: 'pending_approval',
  submittedAt: '2026-10-26T12:00:00Z',
  isNew: seeded,
  form: seeded ? form : undefined,
  answers: seeded
    ? [
        { fieldId: 'motivation', value: { case: 'text', value: 'Хочу поработать над проектом с командой.' } },
        { fieldId: 'experience', value: { case: 'selection', value: { optionIds: ['some'] } } },
        { fieldId: 'portfolio', value: { case: 'files', value: { uploadIds: ['portfolio-file'] } } }
      ]
    : [],
  attachments: seeded
    ? [{ id: 'portfolio-file', fieldId: 'portfolio', fileName: 'portfolio.pdf', contentType: 'application/pdf' }]
    : []
})
let reviewAttempts = 0
const writes: string[] = []
const recordWrite = (operation: string) => {
  writes.push(operation)
  document.documentElement.dataset.writes = JSON.stringify(writes)
}
const transport = createRouterTransport(({ service }) => {
  service(EventsService, {
    getEvent: () => ({
      event: {
        id: 42,
        title: 'Registration workspace event',
        status: EventStatus.CANCELLED,
        startTime: '2027-05-01T12:00:00Z',
        endTime: '2027-05-01T13:00:00Z'
      },
      canEdit: params.get('permission') === 'edit',
      canManageRegistrations: params.get('permission') === 'manage'
    })
  })
  service(EventRegistrationsService, {
    getEventRegistrations: () => ({
      registrations: params.has('statuses')
        ? [
            { id: 1, eventId: 42, userId: 1, status: RegistrationStatus.PENDING_APPROVAL },
            { id: 2, eventId: 42, userId: 2, status: RegistrationStatus.REJECTED }
          ]
        : []
    })
  })
  service(sundayopenServices.SundayOpenService, {
    getExternalRegistrations: () => ({
      total: 2,
      registrations: [
        {
          id: 1,
          eventId: 42,
          fullName: 'Pending visitor',
          status: sundayopenModels.ExternalRegistrationStatus.PENDING_APPROVAL
        },
        {
          id: 2,
          eventId: 42,
          fullName: 'Rejected visitor',
          status: sundayopenModels.ExternalRegistrationStatus.REJECTED
        }
      ]
    })
  })
  service(EventAttendanceService, { getEventAttendance: () => ({}) })
  service(UsersService, {
    getMe: () => ({ user: { id: 1, username: 'Organizer', email: 'organizer@example.test' } }),
    listUsers: () => ({ users: [] })
  })
  service(RegistrationFormsService, {
    getFormDraft: () => {
      calls.push('draft')
      window.dispatchEvent(new Event('workspace-request'))
      return {
        form,
        publishedForm: published,
        canEdit: params.has('design') || !params.has('crm'),
        canManageRegistrations: true
      }
    },
    saveFormDraft: (input) => {
      if (!input.form) throw new ConnectError('Form required', Code.InvalidArgument)
      if (input.expectedRevision !== form.revision) throw new ConnectError('Revision changed', Code.Aborted)
      form = { ...input.form, revision: form.revision + 1 }
      recordWrite('save')
      return { form }
    },
    publishRegistrationForm: (input) => {
      if (input.expectedRevision !== form.revision) throw new ConnectError('Revision changed', Code.Aborted)
      published = { ...form, versionId: `published-${form.revision}` }
      recordWrite('publish')
      return { form: published }
    },
    listEventResponses: () => ({
      responses: seeded || params.has('crm') ? [response] : [],
      total: seeded || params.has('crm') ? 1 : 0,
      unreadCount: response.isNew ? 1 : 0,
      latestResponseId: response.responseId
    }),
    getEventResponse: () => ({ response }),
    getFileDownload: () => ({ url: `${location.origin}/fixture-portfolio.pdf` }),
    exportEventResponses: () => ({
      csv: 'name,email\nАлия Садыкова,aliya@example.test',
      fileName: 'registrations.csv'
    }),
    markResponsesSeen: () => {
      response = { ...response, isNew: false }
      recordWrite('seen')
      return {}
    },
    reviewEventRegistration: (input) => {
      reviewAttempts += 1
      if (params.has('reviewError') && reviewAttempts === 1) throw new ConnectError('Try again', Code.Unavailable)
      response = {
        ...response,
        status:
          input.action === RegistrationReviewAction.APPROVE
            ? 'registered'
            : input.action === RegistrationReviewAction.REOPEN
              ? 'pending_approval'
              : 'rejected'
      }
      recordWrite('review')
      return { response }
    },
    saveResponseNote: (input) => {
      response = { ...response, notes: input.notes }
      recordWrite('note')
      return { response }
    }
  })
})

function Harness() {
  const auth = useAuth()
  return (
    <>
      <button type='button' onClick={() => void auth.checkSession()}>
        Refresh account
      </button>
      <output data-testid='account'>{auth.session?.identity?.id ?? 'signed-out'}</output>
      {params.has('header') ? (
        <EventDetailsModalControlled eventId={42} onClose={() => {}} />
      ) : (
        <RegistrationWorkspacePage eventId={42} />
      )}
    </>
  )
}

window.addEventListener('workspace-request', () => {
  document.documentElement.dataset.draftRequests = String(calls.length)
})

createRoot(document.getElementById('root')!).render(
  <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
    <TransportProvider transport={transport}>
      {params.has('design') ? (
        <main className='rf-workspace-page'>
          <a href='/events'>К мероприятию</a>
          <RegistrationWorkspace eventId={42} client={createClient(RegistrationFormsService, transport)} />
        </main>
      ) : (
        <AuthProvider>
          <Harness />
        </AuthProvider>
      )}
    </TransportProvider>
  </QueryClientProvider>
)
