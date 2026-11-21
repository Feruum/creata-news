import { type Event, type EventBadge, EventBadgeStatus } from '@studyverse/proto'

type EventBadgeFields = Pick<Event, 'socGpaBadge' | 'irosBadge'>

export function isBadgeRequested(badge: EventBadge | undefined): boolean {
  return badge?.status === EventBadgeStatus.PENDING || badge?.status === EventBadgeStatus.APPROVED
}

export function getEventBadgeEntries(event: EventBadgeFields) {
  return [
    { key: 'socGpa', label: 'SocGpa', badge: event.socGpaBadge },
    { key: 'iros', label: 'iRos', badge: event.irosBadge }
  ] as const
}

export function isVisibleEventBadge(badge: EventBadge | undefined, showRequests: boolean) {
  return (
    badge?.status === EventBadgeStatus.APPROVED ||
    (showRequests && (badge?.status === EventBadgeStatus.PENDING || badge?.status === EventBadgeStatus.REJECTED))
  )
}

export function getBadgeStatusLabel(status: EventBadgeStatus | undefined) {
  switch (status) {
    case EventBadgeStatus.PENDING:
      return 'Pending approval'
    case EventBadgeStatus.APPROVED:
      return 'Approved'
    case EventBadgeStatus.REJECTED:
      return 'Rejected'
    default:
      return 'Not requested'
  }
}

export function buildBadgeRequestChanges(
  values: { socGpaRequested: boolean; irosRequested: boolean },
  dirtyFields: { socGpaRequested?: boolean; irosRequested?: boolean }
) {
  return {
    ...(dirtyFields.socGpaRequested ? { socGpaRequested: values.socGpaRequested } : {}),
    ...(dirtyFields.irosRequested ? { irosRequested: values.irosRequested } : {})
  }
}
