import { TrashIcon } from 'lucide-react'
import { toast } from 'sonner'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger
} from '@/shared/ui/components/alert-dialog'
import { Button } from '@/shared/ui/components/button'

interface DeleteEventDialogProps {
  eventId: number
}

export default function DeleteEventDialog({ eventId }: DeleteEventDialogProps) {
  const { removeEvent } = useCalendar()

  const deleteEvent = () => {
    try {
      removeEvent(eventId)
      toast.success('Event deleted successfully.')
    } catch {
      toast.error('Error deleting event.')
    }
  }

  if (!eventId) {
    return null
  }

  return (
    <AlertDialog>
      <AlertDialogTrigger render={<Button variant='destructive' />}>
        <TrashIcon />
        Delete
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete your event and remove event data from our
            servers.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={deleteEvent}>Continue</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
