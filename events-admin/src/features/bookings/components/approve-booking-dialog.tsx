import type { BookingRequest } from '@studyverse/proto'
import { CheckCircleIcon, LinkIcon, Loader2Icon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { useApproveBookingRequest } from '@/features/bookings'
import { OrganizationInfoModal } from '@/features/organizations'
import { UserProfileModal } from '@/features/users'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'

interface ApproveBookingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: BookingRequest | null
}

export function ApproveBookingDialog({ open, onOpenChange, request }: ApproveBookingDialogProps) {
  const [eventId, setEventId] = useState('')

  useEffect(() => {
    if (request?.eventId) {
      setEventId(request.eventId.toString())
    } else {
      setEventId('')
    }
  }, [request?.eventId])

  const mutation = useApproveBookingRequest()

  const handleApprove = async () => {
    if (!request) return

    try {
      await mutation.mutateAsync({
        id: request.id,
        eventId: eventId ? parseInt(eventId, 10) : undefined
      })
      toast.success('Booking request approved')
      onOpenChange(false)
      setEventId('')
    } catch (error) {
      toast.error(error instanceof Error ? `Failed to approve: ${error.message}` : 'Failed to approve')
    }
  }

  if (!request) return null

  const hasLinkedEvent = !!request.eventId

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <CheckCircleIcon className='h-5 w-5 text-emerald-600' />
            Approve Booking Request
          </DialogTitle>
          <DialogDescription>Approve this room booking request for "{request.title}"</DialogDescription>
        </DialogHeader>

        <div className='space-y-4 py-4'>
          <div className='rounded-lg bg-muted/50 p-3 text-sm space-y-1'>
            <p>
              <strong>Room:</strong> {request.room?.name || request.roomId}
            </p>
            <p>
              <strong>Requester:</strong>{' '}
              <UserProfileModal userId={request.requesterId}>
                <button type='button' className='text-primary hover:underline cursor-pointer'>
                  {request.requesterName || 'Unknown'}
                </button>
              </UserProfileModal>
            </p>
            {request.organizationName && request.organizationId && (
              <p>
                <strong>Organization:</strong>{' '}
                <OrganizationInfoModal organizationId={request.organizationId}>
                  <button type='button' className='text-primary hover:underline cursor-pointer'>
                    {request.organizationName}
                  </button>
                </OrganizationInfoModal>
              </p>
            )}
          </div>

          <div className='space-y-2'>
            <div className='flex items-center gap-2'>
              <Label htmlFor='eventId'>Link to Event</Label>
              {hasLinkedEvent && (
                <Badge variant='outline' className='gap-1 bg-emerald-50 text-emerald-700 border-emerald-200'>
                  <LinkIcon className='h-3 w-3' />
                  Linked
                </Badge>
              )}
            </div>
            <Input
              id='eventId'
              type='number'
              placeholder={hasLinkedEvent ? 'Event ID (already linked)' : 'Enter event ID to link this booking'}
              value={eventId}
              onChange={(e) => setEventId(e.target.value)}
            />
            <p className='text-xs text-muted-foreground'>
              {hasLinkedEvent
                ? 'This booking request is already linked to an event. You can change the event ID if needed.'
                : 'Optionally link this booking to an existing event.'}
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleApprove} disabled={mutation.isPending} className='bg-emerald-600 hover:bg-emerald-700'>
            {mutation.isPending && <Loader2Icon className='mr-2 h-4 w-4 animate-spin' />}
            Approve
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
