import { type Room, RoomType } from '@studyverse/proto'
import { AlertCircleIcon, ExternalLinkIcon, Loader2Icon, MapIcon, SearchIcon, XIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useRooms } from '@/features/events'
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { ScrollArea } from '@/shared/ui/components/scroll-area'
import { cn } from '@/shared/ui/lib/utils'

// Map proto RoomType enum to UI-friendly type strings and colors
const ROOM_TYPE_MAP: Partial<Record<RoomType, { label: string; colorClass: string }>> = {
  [RoomType.UNSPECIFIED]: { label: 'other', colorClass: 'bg-gray-100 text-gray-700 border-gray-200' },
  [RoomType.CLASSROOM]: { label: 'classroom', colorClass: 'bg-blue-100 text-blue-700 border-blue-200' },
  [RoomType.LABORATORY]: { label: 'laboratory', colorClass: 'bg-purple-100 text-purple-700 border-purple-200' },
  [RoomType.LIBRARY]: { label: 'library', colorClass: 'bg-amber-100 text-amber-700 border-amber-200' },
  [RoomType.OFFICE]: { label: 'office', colorClass: 'bg-slate-100 text-slate-700 border-slate-200' },
  [RoomType.CAFETERIA]: { label: 'cafeteria', colorClass: 'bg-orange-100 text-orange-700 border-orange-200' },
  [RoomType.GYM]: { label: 'gym', colorClass: 'bg-green-100 text-green-700 border-green-200' },
  [RoomType.AUDITORIUM]: { label: 'auditorium', colorClass: 'bg-indigo-100 text-indigo-700 border-indigo-200' },
  [RoomType.STUDY_ROOM]: { label: 'study room', colorClass: 'bg-cyan-100 text-cyan-700 border-cyan-200' },
  [RoomType.COMMON_AREA]: { label: 'common area', colorClass: 'bg-teal-100 text-teal-700 border-teal-200' },
  [RoomType.MEDICAL]: { label: 'medical', colorClass: 'bg-red-100 text-red-700 border-red-200' },
  [RoomType.OTHER]: { label: 'other', colorClass: 'bg-gray-100 text-gray-700 border-gray-200' }
}

function getRoomTypeInfo(type: RoomType) {
  return ROOM_TYPE_MAP[type] ?? ROOM_TYPE_MAP[RoomType.OTHER]
}

interface RoomSelectionModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSelect: (room: { id: string; name: string; floor: number }) => void
  selectedRoomId?: string
}

export function RoomSelectionModal({ open, onOpenChange, onSelect, selectedRoomId }: RoomSelectionModalProps) {
  const [search, setSearch] = useState('')
  const [selectedFloor, setSelectedFloor] = useState<number | null>(null)

  // Fetch rooms from the API - only bookable and active rooms
  const { data, isLoading, isError, error } = useRooms({
    bookableOnly: true,
    page: 1,
    limit: 500 // Fetch all bookable rooms
  })

  const rooms = data?.rooms ?? []

  // Get unique floors from the data for floor filter buttons
  const availableFloors = [...new Set(rooms.map((room: Room) => room.floor))] as number[]
  availableFloors.sort((a, b) => a - b)

  const filteredRooms = rooms.filter((room: Room) => {
    const matchesSearch =
      !search ||
      room.name.toLowerCase().includes(search.toLowerCase()) ||
      room.id.toLowerCase().includes(search.toLowerCase()) ||
      room.roomNumber.toLowerCase().includes(search.toLowerCase()) ||
      room.block.toLowerCase().includes(search.toLowerCase())

    const matchesFloor = selectedFloor === null || room.floor === selectedFloor

    return matchesSearch && matchesFloor
  })

  const roomsByFloor = filteredRooms.reduce<Record<number, Room[]>>((acc: Record<number, Room[]>, room: Room) => {
    if (!acc[room.floor]) {
      acc[room.floor] = []
    }
    acc[room.floor].push(room)
    return acc
  }, {})

  const handleSelect = (room: Room) => {
    onSelect({ id: room.id, name: room.name, floor: room.floor })
    onOpenChange(false)
  }

  const openMapApp = () => {
    window.open('http://localhost:3000', '_blank', 'noopener,noreferrer')
  }

  useEffect(() => {
    if (!open) {
      setSearch('')
      setSelectedFloor(null)
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-w-2xl max-h-[80vh] flex flex-col'>
        <DialogHeader>
          <DialogTitle className='flex items-center gap-2'>
            <MapIcon className='h-5 w-5' />
            Select Campus Room
          </DialogTitle>
          <DialogDescription>Choose a room from the AITU campus or browse the interactive map</DialogDescription>
        </DialogHeader>

        <div className='space-y-3'>
          <div className='relative'>
            <SearchIcon className='absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground' />
            <Input
              placeholder='Search rooms by name, number, or block...'
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className='pl-9'
            />
            {search && (
              <button
                type='button'
                onClick={() => setSearch('')}
                className='absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground'
              >
                <XIcon className='h-4 w-4' />
              </button>
            )}
          </div>

          <div className='flex items-center gap-2'>
            <span className='text-sm text-muted-foreground'>Floor:</span>
            <div className='flex gap-1'>
              <Button
                variant={selectedFloor === null ? 'secondary' : 'ghost'}
                size='sm'
                onClick={() => setSelectedFloor(null)}
                className='h-7 px-2 text-xs'
              >
                All
              </Button>
              {(availableFloors as number[]).map((floor) => (
                <Button
                  key={floor}
                  variant={selectedFloor === floor ? 'secondary' : 'ghost'}
                  size='sm'
                  onClick={() => setSelectedFloor(floor)}
                  className='h-7 px-2 text-xs'
                >
                  {floor}
                </Button>
              ))}
            </div>
          </div>
        </div>

        <ScrollArea className='flex-1 -mx-6 px-6'>
          {isLoading ? (
            <div className='flex flex-col items-center justify-center py-12 text-muted-foreground'>
              <Loader2Icon className='h-8 w-8 animate-spin mb-3' />
              <p>Loading rooms...</p>
            </div>
          ) : isError ? (
            <div className='flex flex-col items-center justify-center py-12 text-destructive'>
              <AlertCircleIcon className='h-8 w-8 mb-3' />
              <p className='font-medium'>Failed to load rooms</p>
              <p className='text-sm text-muted-foreground mt-1'>{error?.message || 'Please try again later'}</p>
            </div>
          ) : (
            <div className='space-y-4 py-2'>
              {Object.entries(roomsByFloor)
                .sort(([a], [b]) => Number(a) - Number(b))
                .map(([floor, floorRooms]) => (
                  <div key={floor}>
                    <h4 className='text-sm font-medium text-muted-foreground mb-2'>Floor {floor}</h4>
                    <div className='grid gap-2'>
                      {(floorRooms as Room[]).map((room: Room) => {
                        const typeInfo = getRoomTypeInfo(room.type) ?? {
                          label: 'other',
                          colorClass: 'bg-gray-100 text-gray-700 border-gray-200'
                        }
                        return (
                          <button
                            key={room.id}
                            type='button'
                            onClick={() => handleSelect(room)}
                            className={cn(
                              'flex items-center justify-between gap-3 p-3 rounded-lg border text-left transition-colors',
                              selectedRoomId === room.id ? 'border-primary bg-primary/5' : 'hover:bg-muted/50'
                            )}
                          >
                            <div className='flex-1 min-w-0'>
                              <div className='flex items-center gap-2'>
                                <span className='font-medium truncate'>{room.name}</span>
                                <Badge variant='outline' className={cn('text-[10px] capitalize', typeInfo.colorClass)}>
                                  {typeInfo.label}
                                </Badge>
                              </div>
                              <div className='text-xs text-muted-foreground mt-0.5'>
                                {room.block} • Room {room.roomNumber}
                                {room.capacity && ` • Capacity: ${room.capacity}`}
                              </div>
                            </div>
                            {selectedRoomId === room.id && (
                              <Badge variant='secondary' className='shrink-0'>
                                Selected
                              </Badge>
                            )}
                          </button>
                        )
                      })}
                    </div>
                  </div>
                ))}

              {filteredRooms.length === 0 && !isLoading && (
                <div className='text-center py-8 text-muted-foreground'>
                  <p>No rooms found matching your search.</p>
                  <p className='text-sm mt-1'>Try a different search term or browse the map.</p>
                </div>
              )}
            </div>
          )}
        </ScrollArea>

        <div className='border-t pt-4 flex items-center justify-between'>
          <p className='text-xs text-muted-foreground'>Can't find the room? Browse the interactive map</p>
          <Button variant='outline' size='sm' onClick={openMapApp}>
            <ExternalLinkIcon className='h-4 w-4 mr-2' />
            Open Map
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
