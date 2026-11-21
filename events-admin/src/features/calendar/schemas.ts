import { z } from 'zod'

export const eventSchema = z
  .object({
    title: z.string().min(1, 'Title is required'),
    description: z.string().min(1, 'Description is required'),
    startDate: z.date({
      error: (issue) => (issue.input === undefined ? 'Start date is required' : undefined)
    }),
    endDate: z.date({
      error: (issue) => (issue.input === undefined ? 'End date is required' : undefined)
    }),
    hasTime: z.boolean().default(true),
    startTime: z.string().optional(),
    endTime: z.string().optional(),
    color: z.enum(['blue', 'green', 'red', 'yellow', 'purple', 'orange', 'gray'], {
      error: (issue) => (issue.input === undefined ? 'Variant is required' : undefined)
    })
  })
  .superRefine((values, context) => {
    if (!values.hasTime) {
      return
    }

    if (!values.startTime) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'Start time is required',
        path: ['startTime']
      })
    }

    if (!values.endTime) {
      context.addIssue({
        code: z.ZodIssueCode.custom,
        message: 'End time is required',
        path: ['endTime']
      })
    }

    if (values.startTime && values.endTime) {
      const [startHour, startMinute] = values.startTime.split(':').map(Number)
      const [endHour, endMinute] = values.endTime.split(':').map(Number)

      const startMillis = new Date(values.startDate).setHours(startHour, startMinute, 0, 0)
      const endMillis = new Date(values.endDate).setHours(endHour, endMinute, 0, 0)

      if (endMillis <= startMillis) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'End datetime must be after start',
          path: ['endDate']
        })
      }
    }
  })

export type TEventFormInput = z.input<typeof eventSchema>
export type TEventFormData = z.infer<typeof eventSchema>
