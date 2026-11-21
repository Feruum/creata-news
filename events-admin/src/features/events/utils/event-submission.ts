import { EventPublishReadinessState, EventStatus, LocationType } from '@studyverse/proto'
import type { CreateEventFormData } from '../schema'

export function normalizeEventRoomId(roomId = ''): string {
  const id = roomId.trim()
  if (/^[cс]1\./iu.test(id)) return id.replace(/^[cс]1\./iu, 'C1.').toUpperCase()
  if (/^[kк]\./iu.test(id)) return id.replace(/^[kк]\./iu, 'K.').toUpperCase()
  if (['ATRIUM', 'COWORKING', 'FULLGYM', 'DININGHALL'].includes(id.toUpperCase())) return id.toUpperCase()
  return id
}

export function eventBookingProposalSignature(
  data: Partial<
    Pick<CreateEventFormData, 'locationType' | 'roomId' | 'hasTime' | 'startDate' | 'endDate' | 'startTime' | 'endTime'>
  >
): string {
  const campus = data.locationType === LocationType.CAMPUS_ROOM || data.locationType === LocationType.HYBRID
  return JSON.stringify([
    data.locationType,
    campus ? normalizeEventRoomId(data.roomId) : undefined,
    data.hasTime,
    data.startDate?.getTime(),
    data.endDate?.getTime(),
    data.hasTime ? data.startTime : undefined,
    data.hasTime ? data.endTime : undefined
  ])
}

type LocationFields = Pick<
  CreateEventFormData,
  | 'locationType'
  | 'roomId'
  | 'roomName'
  | 'floor'
  | 'externalAddress'
  | 'googleMapsUrl'
  | 'onlineUrl'
  | 'onlinePlatform'
>

export function buildEventLocation(data: LocationFields) {
  const campus = data.locationType === LocationType.CAMPUS_ROOM || data.locationType === LocationType.HYBRID
  const online = data.locationType === LocationType.ONLINE || data.locationType === LocationType.HYBRID
  return {
    type: data.locationType,
    ...(campus && data.roomId?.trim() ? { roomId: normalizeEventRoomId(data.roomId) } : {}),
    ...(campus && data.roomName?.trim() ? { roomName: data.roomName.trim() } : {}),
    ...(campus && data.floor ? { floor: data.floor } : {}),
    ...(data.locationType === LocationType.EXTERNAL && data.externalAddress?.trim()
      ? { externalAddress: data.externalAddress.trim() }
      : {}),
    ...(data.locationType === LocationType.EXTERNAL && data.googleMapsUrl?.trim()
      ? { googleMapsUrl: data.googleMapsUrl.trim() }
      : {}),
    ...(online && data.onlineUrl?.trim() ? { onlineUrl: data.onlineUrl.trim() } : {}),
    ...(online && data.onlinePlatform ? { onlinePlatform: data.onlinePlatform } : {})
  }
}

export function buildEventSchedule(
  data: Pick<CreateEventFormData, 'startDate' | 'endDate' | 'hasTime' | 'startTime' | 'endTime'>
) {
  const start = new Date(data.startDate)
  const end = new Date(data.endDate)
  const [startHour, startMinute] = (data.hasTime ? data.startTime! : '00:00').split(':').map(Number)
  const [endHour, endMinute] = (data.hasTime ? data.endTime! : '23:59').split(':').map(Number)
  start.setHours(startHour!, startMinute!, 0, 0)
  end.setHours(endHour!, endMinute!, 0, 0)
  return { startTime: start.toISOString(), endTime: end.toISOString() }
}

type BookingFields = {
  startTime: string
  endTime?: string
  structuredLocation?: { type: LocationType; roomId?: string }
}

export function needsPublishedBookingCheck(current: BookingFields & { status: EventStatus }, proposal: BookingFields) {
  const campus =
    proposal.structuredLocation?.type === LocationType.CAMPUS_ROOM ||
    proposal.structuredLocation?.type === LocationType.HYBRID
  return (
    current.status === EventStatus.PUBLISHED &&
    (current.structuredLocation?.type !== proposal.structuredLocation?.type ||
      (campus &&
        normalizeEventRoomId(current.structuredLocation?.roomId) !==
          normalizeEventRoomId(proposal.structuredLocation?.roomId)) ||
      Date.parse(current.startTime) !== Date.parse(proposal.startTime) ||
      Date.parse(current.endTime || current.startTime) !== Date.parse(proposal.endTime || proposal.startTime))
  )
}

export function canRequestBooking(state?: EventPublishReadinessState): boolean {
  return (
    state === EventPublishReadinessState.ROOM_BOOKING_MISSING ||
    state === EventPublishReadinessState.ROOM_BOOKING_REJECTED ||
    state === EventPublishReadinessState.ROOM_BOOKING_CANCELLED
  )
}

export function readinessMessage(
  readiness: { state: EventPublishReadinessState; canPublishNow?: boolean },
  publishedEdit = true
): string {
  switch (readiness.state) {
    case EventPublishReadinessState.READY:
    case EventPublishReadinessState.NO_ROOM_APPROVAL_REQUIRED:
      return publishedEdit
        ? 'Room approval is ready. Save Changes to apply your edits.'
        : 'This draft is ready to publish.'
    case EventPublishReadinessState.WAITING_ROOM_APPROVAL:
      return publishedEdit
        ? 'Waiting for room approval. Your event has not changed. Save again after approval.'
        : 'Waiting for room approval before publishing.'
    case EventPublishReadinessState.ROOM_BOOKING_REJECTED:
      return 'The room booking was rejected. You can submit a new request or choose another room or time.'
    case EventPublishReadinessState.ROOM_BOOKING_CANCELLED:
      return 'The room booking was cancelled. Submit a new request before continuing.'
    case EventPublishReadinessState.ROOM_BOOKING_MISSING:
      return 'An approved room booking is required for this room and time. Request room approval to continue.'
    default:
      return readiness.canPublishNow ? 'Ready to continue.' : 'Room approval is required before continuing.'
  }
}
