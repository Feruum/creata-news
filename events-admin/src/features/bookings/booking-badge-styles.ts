import { BookingPriority, BookingRequestStatus } from '@studyverse/proto'

type BadgeStyle = {
  label: string
  className: string
}

export const BOOKING_STATUS_BADGE_STYLES: Record<BookingRequestStatus, BadgeStyle> = {
  [BookingRequestStatus.UNSPECIFIED]: {
    label: 'Unknown',
    className: 'border-border bg-muted/50 text-muted-foreground'
  },
  [BookingRequestStatus.PENDING]: {
    label: 'Pending',
    className: 'border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300'
  },
  [BookingRequestStatus.APPROVED]: {
    label: 'Approved',
    className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
  },
  [BookingRequestStatus.REJECTED]: {
    label: 'Rejected',
    className: 'border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300'
  },
  [BookingRequestStatus.CANCELLED]: {
    label: 'Cancelled',
    className: 'border-slate-500/25 bg-slate-500/10 text-slate-700 dark:text-slate-300'
  }
}

export const BOOKING_PRIORITY_BADGE_STYLES: Record<BookingPriority, BadgeStyle> = {
  [BookingPriority.UNSPECIFIED]: {
    label: 'Unknown',
    className: 'border-border bg-muted/50 text-muted-foreground'
  },
  [BookingPriority.ADMINISTRATIVE]: {
    label: 'Admin',
    className: 'border-fuchsia-500/25 bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-300'
  },
  [BookingPriority.ACADEMIC]: {
    label: 'Academic',
    className: 'border-blue-500/25 bg-blue-500/10 text-blue-700 dark:text-blue-300'
  },
  [BookingPriority.STUDENT_CLUB]: {
    label: 'Club',
    className: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
  },
  [BookingPriority.INDIVIDUAL]: {
    label: 'Personal',
    className: 'border-slate-500/25 bg-slate-500/10 text-slate-700 dark:text-slate-300'
  }
}

export function getBookingStatusBadgeStyle(status: BookingRequestStatus): BadgeStyle {
  return BOOKING_STATUS_BADGE_STYLES[status] || BOOKING_STATUS_BADGE_STYLES[BookingRequestStatus.UNSPECIFIED]
}

export function getBookingPriorityBadgeStyle(priority: BookingPriority): BadgeStyle {
  return BOOKING_PRIORITY_BADGE_STYLES[priority] || BOOKING_PRIORITY_BADGE_STYLES[BookingPriority.UNSPECIFIED]
}
