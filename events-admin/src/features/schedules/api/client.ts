import { z } from 'zod'
import { isNetworkFailure } from '@/shared/lib/offline-interceptor'
import {
  browserIsOnline,
  clearPrivateSnapshots,
  getOfflineState,
  markOffline,
  OfflineReadUnavailable,
  offlineSnapshots,
  recordOfflineUpdate,
  requireVerifiedOnlineWrite
} from '@/shared/lib/offline-runtime'
import type { ReviewDecision, SubmissionDetails, SubmissionFilters, SubmissionList, SubmissionSummary } from '../types'

const endpoint = '/api/schedule/admin/submissions'
const dateTime = z.string().datetime({ offset: true })
const settingsSchema = z.object({
  studyYear: z.number().int(),
  term: z.number().int(),
  firstWeekStart: z.string().regex(/^\d{4}-\d{2}-\d{2}$/)
})
const lessonSchema = z.object({
  id: z.string(),
  title: z.string(),
  start: dateTime,
  end: dateTime,
  teacher: z.string(),
  lessonType: z.string(),
  location: z.string(),
  online: z.boolean()
})
const summarySchema = z.object({
  id: z.string(),
  groupName: z.string(),
  authorId: z.string(),
  contentHash: z.string(),
  status: z.enum(['pending', 'approved', 'rejected', 'superseded']),
  createdAt: dateTime,
  reviewedAt: dateTime.nullable(),
  reviewerId: z.string().nullable(),
  rejectionReason: z.string().nullable()
})
const listSchema = z.object({ submissions: z.array(summarySchema), nextCursor: z.string().nullable() })
const detailsSchema = z.object({
  submission: summarySchema.extend({ settings: settingsSchema, events: z.array(lessonSchema) }),
  published: z
    .object({
      groupName: z.string(),
      settings: settingsSchema,
      events: z.array(lessonSchema),
      updatedAt: dateTime
    })
    .nullable(),
  diff: z.object({ added: z.array(lessonSchema), removed: z.array(lessonSchema) })
})

function messageForStatus(status: number): string {
  if (status === 401) return 'Your session has expired. Sign in again to review schedules.'
  if (status === 403) return 'Admin access is required to review schedules.'
  if (status === 404) return 'This schedule submission is no longer available. Refresh the list.'
  if (status === 409) return 'This schedule has already been reviewed or replaced. Review the refreshed details.'
  return 'The schedule request could not be completed. Please try again.'
}

export class ScheduleApiError extends Error {
  readonly status: number

  constructor(status: number) {
    super(messageForStatus(status))
    this.name = 'ScheduleApiError'
    this.status = status
  }
}

async function request<T>(query: URLSearchParams, schema: z.ZodType<T>, init: RequestInit = {}): Promise<T> {
  const suffix = query.size ? `?${query.toString()}` : ''
  const isRead = !init.method || init.method === 'GET'
  const method = `schedule.admin/${query.has('id') ? 'GetSubmission' : 'ListSubmissions'}`
  const input = Object.fromEntries(query.entries())
  let lease = offlineSnapshots.lease()
  const isCurrentRequest = () => (lease ? offlineSnapshots.isCurrent(lease) : offlineSnapshots.lease() === null)
  const readSaved = async () => {
    const saved = await offlineSnapshots.read(method, input)
    if (!isCurrentRequest()) throw new Error('The signed-in account changed.')
    if (!saved) throw new OfflineReadUnavailable()
    recordOfflineUpdate(saved.updatedAt)
    return schema.parse(saved.data)
  }
  if (!isRead) lease = await requireVerifiedOnlineWrite()
  if (isRead && (!browserIsOnline() || getOfflineState().mode === 'offline-readonly')) return readSaved()
  try {
    const response = await fetch(`${endpoint}${suffix}`, {
      ...init,
      credentials: 'include',
      cache: 'no-store'
    })
    if (!isCurrentRequest()) throw new Error('The signed-in account changed.')
    if (response.status === 401) await clearPrivateSnapshots()
    if (!response.ok) throw new ScheduleApiError(response.status)

    const data: unknown = await response.json()
    if (!isCurrentRequest()) throw new Error('The signed-in account changed.')
    const parsed = schema.safeParse(data)
    if (!parsed.success) throw new Error('Schedule details could not be loaded. Please refresh and try again.')
    if (isRead && getOfflineState().mode === 'online') await offlineSnapshots.save(lease, method, input, parsed.data)
    if (!isCurrentRequest()) throw new Error('The signed-in account changed.')
    return parsed.data
  } catch (error) {
    if (!isCurrentRequest()) throw new Error('The signed-in account changed.')
    if (isRead && isNetworkFailure(error)) {
      markOffline()
      return readSaved()
    }
    throw error
  }
}

export function listSubmissions(
  filters: SubmissionFilters,
  cursor?: string,
  signal?: AbortSignal
): Promise<SubmissionList> {
  const query = new URLSearchParams({ status: filters.status })
  if (filters.group?.trim()) query.set('group', filters.group.trim())
  if (cursor) query.set('cursor', cursor)
  return request(query, listSchema, { signal })
}

export function getSubmission(id: string, signal?: AbortSignal): Promise<SubmissionDetails> {
  return request(new URLSearchParams({ id }), detailsSchema, { signal })
}

export async function reviewSubmission(input: ReviewDecision): Promise<SubmissionSummary> {
  const reason = input.reason?.trim()
  if (input.decision === 'reject' && !reason) throw new Error('A rejection reason is required.')
  const body: ReviewDecision = { id: input.id, decision: input.decision }
  if (input.decision === 'reject') body.reason = reason

  return request(new URLSearchParams(), summarySchema, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  })
}

export function scheduleErrorMessage(error: unknown): string {
  if (error instanceof OfflineReadUnavailable) return error.message
  if (error instanceof ScheduleApiError) return error.message
  return 'Schedules could not be loaded. Please try again.'
}
