import type { BookingRequest, Room } from '@studyverse/proto'
import { CheckIcon, Loader2Icon, MapPinIcon, SparklesIcon, UsersIcon, XCircleIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { useAlternativeRooms, useRejectBookingRequest } from '@/features/bookings'

import { Button } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/shared/ui/components/dialog'
import { Label } from '@/shared/ui/components/label'
import { Skeleton } from '@/shared/ui/components/skeleton'
import { Textarea } from '@/shared/ui/components/textarea'
import { cn } from '@/shared/ui/lib/utils'

interface RejectBookingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: BookingRequest | null
}

export function RejectBookingDialog({ open, onOpenChange, request }: RejectBookingDialogProps) {
  const [reason, setReason] = useState('')
  const [selectedRoomIds, setSelectedRoomIds] = useState<string[]>([])
  const [showSuggestions, setShowSuggestions] = useState(false)

  const { data: alternativesData, isLoading: loadingAlternatives } = useAlternativeRooms({
    startTime: request?.startTime ?? '',
    endTime: request?.endTime ?? '',
    minCapacity: request?.room?.capacity,
    preferredFloor: request?.room?.floor,
    enabled: open && Boolean(request?.startTime) && showSuggestions
  })

  useEffect(() => {
    if (!open) {
      setReason('')
      setSelectedRoomIds([])
      setShowSuggestions(false)
    }
  }, [open])

  const mutation = useRejectBookingRequest()

  const handleReject = async () => {
    if (!request) return

    if (!reason.trim()) {
      toast.error('Please provide a reason for rejection')
      return
    }

    try {
      await mutation.mutateAsync({
        id: request.id,
        reason: reason.trim(),
        suggestedRoomIds: selectedRoomIds
      })
      toast.success('Booking request rejected')
      onOpenChange(false)
    } catch (error) {
      toast.error(error instanceof Error ? `Failed to reject: ${error.message}` : 'Failed to reject')
    }
  }

  const toggleRoom = (roomId: string) => {
    setSelectedRoomIds((prev) => (prev.includes(roomId) ? prev.filter((id) => id !== roomId) : [...prev, roomId]))
  }

  if (!request) return null

  const alternatives = alternativesData?.availableRooms ?? []
  const filteredAlternatives = alternatives.filter((room: Room) => room.id !== request.roomId)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <XCircleIcon className='h-5 w-5 text-red-600' />
            Reject Booking Request
          </DialogTitle>
          <DialogDescription>Reject this room booking request for "{request.title}"</DialogDescription>
        </DialogHeader>

        <div className='space-y-4 py-4'>
          <div className='rounded-lg bg-muted/50 p-3 text-sm space-y-1'>
            <p>
              <strong>Room:</strong> {request.room?.name || 'Unknown'}
            </p>
            <p>
              <strong>Requester:</strong> {request.requesterName || 'Unknown'}
            </p>
            {request.organizationName && (
              <p>
                <strong>Organization:</strong> {request.organizationName}
              </p>
            )}
          </div>

          <div className='space-y-2'>
            <Label htmlFor='reason'>Reason for Rejection *</Label>
            <Textarea
              id='reason'
              placeholder='Please explain why this request is being rejected...'
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
            />
            <p className='text-xs text-muted-foreground'>The requester will be notified with this reason.</p>
          </div>

          <div className='space-y-3'>
            <div className='flex items-center justify-between'>
              <Label className='flex items-center gap-2'>
                <SparklesIcon className='h-4 w-4 text-blue-500' />
                Suggest Alternative Rooms
              </Label>
              <Button
                variant='ghost'
                size='sm'
                onClick={() => setShowSuggestions(!showSuggestions)}
                className='text-xs'
              >
                {showSuggestions ? 'Hide' : 'Show'}
              </Button>
            </div>

            {showSuggestions && (
              <div className='rounded-lg border p-3 space-y-2 max-h-48 overflow-y-auto'>
                {loadingAlternatives ? (
                  <div className='space-y-2'>
                    <Skeleton className='h-12 w-full' />
                    <Skeleton className='h-12 w-full' />
                  </div>
                ) : filteredAlternatives.length === 0 ? (
                  <p className='text-sm text-muted-foreground text-center py-2'>
                    No alternative rooms available for this time slot.
                  </p>
                ) : (
                  filteredAlternatives.map((room: Room) => (
                    <AlternativeRoomOption
                      key={room.id}
                      room={room}
                      selected={selectedRoomIds.includes(room.id)}
                      onToggle={() => toggleRoom(room.id)}
                    />
                  ))
                )}
                {selectedRoomIds.length > 0 && (
                  <p className='text-xs text-muted-foreground pt-2 border-t'>
                    {selectedRoomIds.length} room{selectedRoomIds.length > 1 ? 's' : ''} will be suggested to the
                    requester.
                  </p>
                )}
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant='outline' onClick={() => onOpenChange(false)} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button variant='destructive' onClick={handleReject} disabled={mutation.isPending || !reason.trim()}>
            {mutation.isPending && <Loader2Icon className='mr-2 h-4 w-4 animate-spin' />}
            Reject
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface AlternativeRoomOptionProps {
  room: Room
  selected: boolean
  onToggle: () => void
}

function AlternativeRoomOption({ room, selected, onToggle }: AlternativeRoomOptionProps) {
  return (
    <label
      htmlFor={`alternative-room-${room.id}`}
      className={cn(
        'flex items-center gap-3 p-2 rounded-md cursor-pointer transition-colors',
        selected ? 'bg-blue-50 border border-blue-200' : 'hover:bg-muted/50 border border-transparent'
      )}
    >
      <Checkbox id={`alternative-room-${room.id}`} checked={selected} onCheckedChange={onToggle} />
      <div className='flex-1 min-w-0'>
        <div className='flex items-center gap-2'>
          <MapPinIcon className='h-3.5 w-3.5 text-muted-foreground shrink-0' />
          <span className='text-sm font-medium truncate'>{room.name}</span>
        </div>
        <div className='flex items-center gap-3 text-xs text-muted-foreground mt-0.5'>
          <span>
            {room.block} • Floor {room.floor}
          </span>
          {room.capacity && (
            <span className='flex items-center gap-1'>
              <UsersIcon className='h-3 w-3' />
              {room.capacity}
            </span>
          )}
        </div>
      </div>
      {selected && <CheckIcon className='h-4 w-4 text-blue-600 shrink-0' />}
    </label>
  )
}
