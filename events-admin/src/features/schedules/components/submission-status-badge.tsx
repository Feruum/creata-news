import { Badge } from '@/shared/ui/components/badge'
import type { SubmissionStatus } from '../types'

const labels: Record<SubmissionStatus, string> = {
  pending: 'Pending',
  approved: 'Approved',
  rejected: 'Rejected',
  superseded: 'Superseded'
}

export function SubmissionStatusBadge({ status }: { status: SubmissionStatus }) {
  return <Badge variant={status === 'rejected' ? 'destructive' : 'outline'}>{labels[status]}</Badge>
}
