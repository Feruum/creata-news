import { LocationType } from '@studyverse/proto'
import { Building2, Globe, Layers, MapIcon, Video } from 'lucide-react'
import { useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { Button } from '@/shared/ui/components/button'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/shared/ui/components/form'
import { Input } from '@/shared/ui/components/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { Switch } from '@/shared/ui/components/switch'
import { Textarea } from '@/shared/ui/components/textarea'
import { cn } from '@/shared/ui/lib/utils'
import { RoomSelectionModal } from './room-selection-modal'

const locationTypeOptions = [
  {
    value: LocationType.CAMPUS_ROOM,
    label: 'Campus Room',
    icon: Building2,
    description: 'AITU campus location'
  },
  {
    value: LocationType.EXTERNAL,
    label: 'External Venue',
    icon: Globe,
    description: 'Off-campus location'
  },
  {
    value: LocationType.ONLINE,
    label: 'Online',
    icon: Video,
    description: 'Virtual meeting'
  },
  {
    value: LocationType.HYBRID,
    label: 'Hybrid',
    icon: Layers,
    description: 'Both in-person and online'
  }
] as const

const floorOptions = [
  { value: '1', label: 'Floor 1' },
  { value: '2', label: 'Floor 2' },
  { value: '3', label: 'Floor 3' }
]

const platformOptions = [
  { value: 'zoom', label: 'Zoom' },
  { value: 'teams', label: 'Microsoft Teams' },
  { value: 'google_meet', label: 'Google Meet' },
  { value: 'other', label: 'Other' }
]

export interface LocationFormData {
  locationType: LocationType
  roomId?: string
  roomName?: string
  floor?: number
  externalAddress?: string
  googleMapsUrl?: string
  onlineUrl?: string
  onlinePlatform?: string
}

interface LocationFormSectionProps {
  form: UseFormReturn<any>
  fieldPrefix?: string
}

export function LocationFormSection({ form, fieldPrefix = '' }: LocationFormSectionProps) {
  const [isRoomModalOpen, setIsRoomModalOpen] = useState(false)
  const getFieldName = (name: string) => (fieldPrefix ? `${fieldPrefix}.${name}` : name) as any

  const locationType = form.watch(getFieldName('locationType')) as LocationType

  const showCampusFields = locationType === LocationType.CAMPUS_ROOM || locationType === LocationType.HYBRID
  const showExternalFields = locationType === LocationType.EXTERNAL
  const showOnlineFields = locationType === LocationType.ONLINE || locationType === LocationType.HYBRID

  const handleRoomSelect = (room: { id: string; name: string; floor: number }) => {
    form.setValue(getFieldName('roomId'), room.id)
    form.setValue(getFieldName('roomName'), room.name)
    form.setValue(getFieldName('floor'), room.floor)
  }

  return (
    <div className='space-y-4'>
      <FormField
        control={form.control}
        name={getFieldName('locationType')}
        render={({ field }) => (
          <FormItem>
            <FormLabel>Location Type</FormLabel>
            <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
              {locationTypeOptions.map((option) => {
                const Icon = option.icon
                const isSelected = field.value === option.value
                return (
                  <button
                    key={option.value}
                    type='button'
                    onClick={() => field.onChange(option.value)}
                    className={cn(
                      'flex flex-col items-center gap-1.5 rounded-lg border p-3 text-center transition-colors',
                      isSelected
                        ? 'border-primary bg-primary/5 text-primary'
                        : 'border-border hover:border-primary/50 hover:bg-muted/50'
                    )}
                  >
                    <Icon className='h-5 w-5' />
                    <span className='text-xs font-medium'>{option.label}</span>
                  </button>
                )
              })}
            </div>
            <FormMessage />
          </FormItem>
        )}
      />

      {showCampusFields && (
        <div className='space-y-3 rounded-lg border border-dashed p-4'>
          <div className='flex items-center justify-between'>
            <p className='text-xs font-medium text-muted-foreground'>Campus Location</p>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => setIsRoomModalOpen(true)}
              className='h-7 text-xs'
            >
              <MapIcon className='h-3.5 w-3.5 mr-1.5' />
              Select on Map
            </Button>
          </div>
          <div className='grid gap-3 sm:grid-cols-2'>
            <FormField
              control={form.control}
              name={getFieldName('roomId')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Room ID</FormLabel>
                  <FormControl>
                    <Input placeholder='e.g., C1.2.321' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name={getFieldName('roomName')}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Room Name</FormLabel>
                  <FormControl>
                    <Input placeholder='e.g., Lecture Hall 321' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name={getFieldName('floor')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Floor</FormLabel>
                <Select
                  onValueChange={(v) => {
                    if (v === null) return
                    return field.onChange(Number(v))
                  }}
                  value={field.value?.toString() ?? ''}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder='Select floor' />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {floorOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={getFieldName('requestRoomBooking')}
            render={({ field }) => (
              <FormItem className='flex flex-row items-center justify-between rounded-lg border p-3'>
                <div className='space-y-0.5'>
                  <FormLabel className='text-sm'>Request room booking</FormLabel>
                  <FormDescription className='text-xs'>
                    Submit a booking request for this room when the event is created
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch checked={field.value} onCheckedChange={field.onChange} />
                </FormControl>
              </FormItem>
            )}
          />
        </div>
      )}

      {showExternalFields && (
        <div className='space-y-3 rounded-lg border border-dashed p-4'>
          <p className='text-xs font-medium text-muted-foreground'>External Venue</p>
          <FormField
            control={form.control}
            name={getFieldName('externalAddress')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Address</FormLabel>
                <FormControl>
                  <Textarea placeholder='Enter the full address of the venue' className='min-h-[80px]' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={getFieldName('googleMapsUrl')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Google Maps Link (optional)</FormLabel>
                <FormControl>
                  <Input placeholder='https://maps.google.com/...' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}

      {showOnlineFields && (
        <div className='space-y-3 rounded-lg border border-dashed p-4'>
          <p className='text-xs font-medium text-muted-foreground'>Online Meeting</p>
          <FormField
            control={form.control}
            name={getFieldName('onlinePlatform')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Platform</FormLabel>
                <Select
                  onValueChange={(value) => {
                    if (value !== null) field.onChange(value)
                  }}
                  value={field.value ?? ''}
                >
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder='Select platform' />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {platformOptions.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name={getFieldName('onlineUrl')}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Meeting URL</FormLabel>
                <FormControl>
                  <Input placeholder='https://zoom.us/j/...' {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}

      <RoomSelectionModal
        open={isRoomModalOpen}
        onOpenChange={setIsRoomModalOpen}
        onSelect={handleRoomSelect}
        selectedRoomId={form.watch(getFieldName('roomId'))}
      />
    </div>
  )
}
