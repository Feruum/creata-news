import { SendIcon } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { useSendTelegramNotification } from '@/features/users'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import { Label } from '@/shared/ui/components/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { Textarea } from '@/shared/ui/components/textarea'

interface SendNotificationDialogProps {
  userIds: number[]
  onSuccess?: () => void
  children?: React.ReactElement
}

const MAX_MESSAGE_LENGTH = 4096

export function SendNotificationDialog({ userIds, onSuccess, children }: SendNotificationDialogProps) {
  const [open, setOpen] = useState(false)
  const [message, setMessage] = useState('')
  const [parseMode, setParseMode] = useState<string>('none')

  const mutation = useSendTelegramNotification()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const data = await mutation.mutateAsync({
        userIds,
        message,
        parseMode: parseMode === 'none' ? undefined : parseMode
      })
      const { sent, failed, skipped } = data
      if (sent > 0) {
        toast.success(
          `Sent ${sent} notification${sent > 1 ? 's' : ''}${failed > 0 ? `, ${failed} failed` : ''}${skipped > 0 ? `, ${skipped} skipped` : ''}`
        )
      } else if (skipped > 0) {
        toast.warning(`No notifications sent. ${skipped} user${skipped > 1 ? 's' : ''} without linked Telegram.`)
      } else {
        toast.error('Failed to send notifications')
      }
      setOpen(false)
      setMessage('')
      setParseMode('none')
      onSuccess?.()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to send notifications')
    }
  }

  const isDisabled = mutation.isPending || message.trim().length === 0 || userIds.length === 0

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          children || (
            <Button variant='outline' size='sm' disabled={userIds.length === 0}>
              <SendIcon className='mr-1.5 h-3.5 w-3.5' />
              Send Notification
            </Button>
          )
        }
      ></DialogTrigger>
      <DialogContent className='sm:max-w-lg'>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Send Telegram Notification</DialogTitle>
            <DialogDescription>
              Send a message to {userIds.length} selected user{userIds.length > 1 ? 's' : ''} via Telegram. Users
              without linked Telegram accounts will be skipped.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='message'>Message</Label>
              <Textarea
                id='message'
                value={message}
                onChange={(e) => setMessage(e.target.value.slice(0, MAX_MESSAGE_LENGTH))}
                placeholder='Enter your notification message...'
                rows={5}
                className='resize-none'
              />
              <div className='flex justify-between text-xs text-muted-foreground'>
                <span>
                  {message.length} / {MAX_MESSAGE_LENGTH}
                </span>
              </div>
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='parseMode'>Formatting</Label>
              <Select
                value={parseMode}
                onValueChange={(value) => {
                  if (value !== null) setParseMode(value)
                }}
              >
                <SelectTrigger id='parseMode'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='none'>Plain Text</SelectItem>
                  <SelectItem value='HTML'>HTML</SelectItem>
                  <SelectItem value='Markdown'>Markdown</SelectItem>
                  <SelectItem value='MarkdownV2'>MarkdownV2</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type='submit' disabled={isDisabled}>
              {mutation.isPending ? 'Sending...' : 'Send'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
