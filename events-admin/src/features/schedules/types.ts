export type SubmissionStatus = 'pending' | 'approved' | 'rejected' | 'superseded'

export interface ScheduleSettings {
  studyYear: number
  term: number
  firstWeekStart: string
}

export interface ScheduleLesson {
  id: string
  title: string
  start: string
  end: string
  teacher: string
  lessonType: string
  location: string
  online: boolean
}

export interface SubmissionSummary {
  id: string
  groupName: string
  authorId: string
  contentHash: string
  status: SubmissionStatus
  createdAt: string
  reviewedAt: string | null
  reviewerId: string | null
  rejectionReason: string | null
}

export interface SubmissionList {
  submissions: SubmissionSummary[]
  nextCursor: string | null
}

export interface SubmissionDetails {
  submission: SubmissionSummary & { settings: ScheduleSettings; events: ScheduleLesson[] }
  published: {
    groupName: string
    settings: ScheduleSettings
    events: ScheduleLesson[]
    updatedAt: string
  } | null
  diff: { added: ScheduleLesson[]; removed: ScheduleLesson[] }
}

export interface ReviewDecision {
  id: string
  decision: 'approve' | 'reject'
  reason?: string
}

export interface SubmissionFilters {
  status: 'pending' | 'all'
  group?: string
}
