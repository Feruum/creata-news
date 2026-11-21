/**
 * @deprecated RESERVED FOR FUTURE USE - See _draft-multi-step-event-form.tsx
 */

import { EventFormat, LocationType } from '@studyverse/proto'
import { z } from 'zod'
import type { Step } from '@/shared/hooks/use-multi-step-form'
import { getPlateTextLength, parsePlateJson } from '@/shared/lib/plate-utils'

// ==========================================================================
// CREATE EVENT FORM SCHEMA
// ==========================================================================

export const createEventSchema = z
  .object({
    hasTime: z.boolean().default(true),
    startTime: z.string().optional(),
    endTime: z.string().optional(),
    title: z
      .string()
      .trim()
      .min(3, 'Title must be at least 3 characters')
      .max(100, 'Title must be less than 100 characters'),
    description: z.any().refine(
      (val) => {
        const nodes: unknown = val
        if (typeof nodes === 'string') {
          try {
            return getPlateTextLength(parsePlateJson(nodes)) >= 10
          } catch {
            return false
          }
        }
        if (!Array.isArray(nodes) || nodes.length === 0) return false
        let textLen = 0
        const walk = (nodes: unknown[]) => {
          for (const node of nodes) {
            if (typeof node === 'object' && node !== null) {
              const n = node as Record<string, unknown>
              if (typeof n.text === 'string') textLen += n.text.length
              if (Array.isArray(n.children)) walk(n.children)
            }
          }
        }
        walk(nodes)
        return textLen >= 10
      },
      { message: 'Описание должно содержать не менее 10 символов' }
    ),
    location: z.string().optional(),
    virtualLink: z.string().url('Please enter a valid URL').optional().or(z.literal('')),
    startDate: z.date({ error: (issue) => (issue.input === undefined ? 'Start date is required' : undefined) }),
    endDate: z.date({ error: (issue) => (issue.input === undefined ? 'End date is required' : undefined) }),
    organizationIds: z.array(z.number().int().positive()).length(1, 'Select one primary hosting organization'),
    categoryId: z.number().int().positive().optional(),
    coverImage: z
      .instanceof(File)
      .refine((file) => file.size > 0, 'Choose a nonempty image')
      .refine((file) => file.size <= 5 * 1024 * 1024, 'Image must be 5 MB or smaller')
      .refine(
        (file) => ['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type),
        'Choose a JPEG, PNG, WebP or GIF image'
      )
      .optional(),
    locationType: z.nativeEnum(LocationType).default(LocationType.CAMPUS_ROOM),
    roomId: z.string().trim().optional(),
    roomName: z.string().trim().optional(),
    floor: z.number().int().min(1).max(3).optional(),
    externalAddress: z.string().trim().optional(),
    googleMapsUrl: z.string().optional(),
    onlineUrl: z.string().optional(),
    onlinePlatform: z.string().optional(),
    requestRoomBooking: z.boolean().default(true),
    socGpaRequested: z.boolean().default(false),
    irosRequested: z.boolean().default(false)
  })
  .superRefine((data, ctx) => {
    const validClock = /^([01]\d|2[0-3]):[0-5]\d$/
    if (data.hasTime) {
      if (!data.startTime) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Start time is required', path: ['startTime'] })
      }
      if (!data.endTime) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'End time is required', path: ['endTime'] })
      }
      for (const field of ['startTime', 'endTime'] as const) {
        if (data[field] && !validClock.test(data[field])) {
          ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Enter a valid time (HH:mm)', path: [field] })
        }
      }
    }

    const start = new Date(data.startDate)
    const end = new Date(data.endDate)

    if (data.hasTime && data.startTime) {
      const [hours, minutes] = data.startTime.split(':').map(Number)
      start.setHours(hours ?? 0, minutes ?? 0, 0, 0)
    } else {
      start.setHours(0, 0, 0, 0)
    }

    if (data.hasTime && data.endTime) {
      const [hours, minutes] = data.endTime.split(':').map(Number)
      end.setHours(hours ?? 0, minutes ?? 0, 0, 0)
    } else {
      end.setHours(23, 59, 0, 0)
    }

    if (!(end > start)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'End date must be after start date', path: ['endDate'] })
    }

    if (data.locationType === LocationType.CAMPUS_ROOM || data.locationType === LocationType.HYBRID) {
      if (!data.roomId && !data.roomName) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Room ID or name is required', path: ['roomId'] })
      }
    }
    if (data.locationType === LocationType.EXTERNAL) {
      if (!data.externalAddress) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Address is required', path: ['externalAddress'] })
      }
      if (data.googleMapsUrl?.trim() && !isWebUrl(data.googleMapsUrl)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Enter a valid HTTP or HTTPS URL',
          path: ['googleMapsUrl']
        })
      }
    }
    if (data.locationType === LocationType.ONLINE || data.locationType === LocationType.HYBRID) {
      if (!data.onlineUrl?.trim()) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Meeting URL is required', path: ['onlineUrl'] })
      } else if (!isWebUrl(data.onlineUrl)) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: 'Enter a valid HTTP or HTTPS URL', path: ['onlineUrl'] })
      }
    }
  })

function isWebUrl(value: string): boolean {
  try {
    const url = new URL(value.trim())
    return (url.protocol === 'https:' || url.protocol === 'http:') && Boolean(url.hostname)
  } catch {
    return false
  }
}

export type CreateEventFormInput = z.input<typeof createEventSchema>
export type CreateEventFormData = z.infer<typeof createEventSchema>

export const defaultValues: Partial<CreateEventFormData> = {
  title: '',
  description: '',
  hasTime: true,
  organizationIds: [],
  categoryId: undefined,
  location: '',
  virtualLink: '',
  locationType: LocationType.CAMPUS_ROOM,
  roomId: '',
  roomName: '',
  floor: undefined,
  externalAddress: '',
  googleMapsUrl: '',
  onlineUrl: '',
  onlinePlatform: '',
  requestRoomBooking: true,
  socGpaRequested: false,
  irosRequested: false
}

// ============================================================================
// MULTI-STEP FORM SCHEMA
// Split schema by steps for partial validation
// ============================================================================

// Step 1: Basic Info
export const basicInfoSchema = z.object({
  title: z.string().min(3, 'Title must be at least 3 characters'),
  description: z.string().min(10, 'Description must be at least 10 characters'),
  format: z.nativeEnum(EventFormat)
})

// Step 2: Schedule & Location
export const scheduleSchema = z.object({
  hasTime: z.boolean().default(true),
  startDate: z.date({ error: (issue) => (issue.input === undefined ? 'Start date is required' : undefined) }),
  startTime: z.string().optional(),
  endDate: z.date({ error: (issue) => (issue.input === undefined ? 'End date is required' : undefined) }),
  endTime: z.string().optional(),
  location: z.string().optional()
})

// Step 3: Settings
export const settingsSchema = z.object({
  capacity: z.coerce.number<string | number>().min(1).max(10000).optional(),
  isPublic: z.boolean().default(true),
  coverImage: z.instanceof(File).optional(),
  categoryId: z.number().optional()
})

// Combined schema for final submission
export const fullEventSchema = basicInfoSchema
  .merge(scheduleSchema)
  .merge(settingsSchema)
  .superRefine((data, ctx) => {
    if (data.hasTime) {
      if (!data.startTime) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Start time is required',
          path: ['startTime']
        })
      }

      if (!data.endTime) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'End time is required',
          path: ['endTime']
        })
      }
    }

    const start = new Date(data.startDate)
    const end = new Date(data.endDate)

    if (data.hasTime && data.startTime) {
      const [hours, minutes] = data.startTime.split(':').map(Number)
      start.setHours(hours ?? 0, minutes ?? 0, 0, 0)
    } else {
      start.setHours(0, 0, 0, 0)
    }

    if (data.hasTime && data.endTime) {
      const [hours, minutes] = data.endTime.split(':').map(Number)
      end.setHours(hours ?? 0, minutes ?? 0, 0, 0)
    } else {
      end.setHours(0, 0, 0, 0)
    }

    if (!(end > start)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'End date must be after start date',
        path: ['endDate']
      })
    }
  })

export type EventFormInput = z.input<typeof fullEventSchema>
export type EventFormData = z.infer<typeof fullEventSchema>

// ============================================================================
// STEP DEFINITIONS
// ============================================================================

export const eventFormSteps: Step[] = [
  {
    id: 'basic-info',
    title: 'Basic Info',
    description: 'Event name and description',
    fields: ['title', 'description', 'format']
  },
  {
    id: 'schedule',
    title: 'Schedule',
    description: 'When and where',
    fields: ['hasTime', 'startDate', 'startTime', 'endDate', 'endTime', 'location']
  },
  {
    id: 'settings',
    title: 'Settings',
    description: 'Capacity and visibility',
    fields: ['capacity', 'isPublic', 'coverImage', 'categoryId']
  }
]
