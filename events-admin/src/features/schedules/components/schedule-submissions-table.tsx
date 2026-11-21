import { formatDate } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/components/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'
import type { SubmissionSummary } from '../types'
import { SubmissionStatusBadge } from './submission-status-badge'

interface ScheduleSubmissionsTableProps {
  submissions: SubmissionSummary[]
  onSelect: (id: string, trigger: HTMLButtonElement) => void
}

export function ScheduleSubmissionsTable({ submissions, onSelect }: ScheduleSubmissionsTableProps) {
  return (
    <div className='min-w-0 rounded-lg border'>
      <Table aria-label='Schedule submissions'>
        <TableHeader>
          <TableRow>
            <TableHead scope='col'>Group</TableHead>
            <TableHead scope='col'>Status</TableHead>
            <TableHead scope='col'>Submitted</TableHead>
            <TableHead scope='col'>Author</TableHead>
            <TableHead scope='col'>Review</TableHead>
            <TableHead scope='col'>
              <span className='sr-only'>Actions</span>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {submissions.map((submission) => (
            <TableRow key={submission.id}>
              <TableCell className='font-medium'>{submission.groupName}</TableCell>
              <TableCell>
                <SubmissionStatusBadge status={submission.status} />
              </TableCell>
              <TableCell className='whitespace-nowrap'>
                <time dateTime={submission.createdAt}>{formatDate(submission.createdAt, true)}</time>
              </TableCell>
              <TableCell className='max-w-52 break-all text-xs'>{submission.authorId}</TableCell>
              <TableCell className='min-w-48 max-w-72 whitespace-normal'>
                {submission.reviewedAt ? (
                  <div className='space-y-1'>
                    <time dateTime={submission.reviewedAt}>{formatDate(submission.reviewedAt, true)}</time>
                    <p className='break-all text-xs text-muted-foreground'>{submission.reviewerId}</p>
                    {submission.rejectionReason && <p className='text-sm'>{submission.rejectionReason}</p>}
                  </div>
                ) : (
                  <span className='text-muted-foreground'>
                    {submission.status === 'pending'
                      ? 'Awaiting review'
                      : submission.status === 'superseded'
                        ? 'Replaced by a newer submission'
                        : '—'}
                  </span>
                )}
              </TableCell>
              <TableCell className='text-right'>
                <Button
                  size='sm'
                  variant='outline'
                  aria-label={`${submission.status === 'pending' ? 'Review' : 'View'} ${submission.groupName}`}
                  onClick={(event) => onSelect(submission.id, event.currentTarget)}
                >
                  {submission.status === 'pending' ? 'Review' : 'View'}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
