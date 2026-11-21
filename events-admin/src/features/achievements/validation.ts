import { AchievementEffect, AchievementRuleKind, AchievementStatus } from '@studyverse/proto'
import { z } from 'zod'

const optionalId = z.number().int().min(0).max(2147483647)
const imageUrl = z.string().refine((value) => !value || /^https?:\/\//i.test(value), 'Use an HTTPS or HTTP image URL')

export const achievementFormSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, 'Title is required')
    .max(200)
    .refine((value) => new TextEncoder().encode(value).length <= 200, 'Title is too long. Please shorten it.'),
  description: z
    .string()
    .max(4000)
    .refine((value) => new TextEncoder().encode(value).length <= 4000, 'Description is too long. Please shorten it.'),
  imageUrl,
  badgeIconUrl: imageUrl,
  backgroundColor: z.string().regex(/^#[0-9a-f]{6}$/i, 'Use a six digit hex color'),
  accentColor: z.string().regex(/^#[0-9a-f]{6}$/i, 'Use a six digit hex color'),
  effect: z.number().int().min(AchievementEffect.NONE).max(AchievementEffect.SPARKLE),
  status: z.number().int().min(AchievementStatus.DRAFT).max(AchievementStatus.ARCHIVED),
  ruleKind: z.nativeEnum(AchievementRuleKind),
  threshold: z.number().int().positive('Threshold must be a positive whole number').max(100000),
  eventId: optionalId,
  organizationId: optionalId,
  categoryId: optionalId
})

export type AchievementFormValues = z.infer<typeof achievementFormSchema>

export function selectedRecipientIds(selection: Record<string, boolean>): number[] {
  return [
    ...new Set(
      Object.entries(selection)
        .filter(([, checked]) => checked)
        .map(([id]) => Number(id))
    )
  ].filter((id) => Number.isSafeInteger(id) && id > 0)
}

export function validateAchievementImage(file: { type: string; size: number }): string | undefined {
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) return 'Choose a PNG, JPEG, or WebP image'
  if (file.size <= 0 || file.size > 5 * 1024 * 1024) return 'Image must be between 1 byte and 5 MB'
}

export function validateAwardAction(recipientIds: number[], reason: string): string | undefined {
  if (!recipientIds.length) return 'Choose at least one recipient'
  if (recipientIds.length > 500) return 'Choose no more than 500 recipients per action'
  if (!reason.trim()) return 'An audit reason is required'
  if (new TextEncoder().encode(reason.trim()).length > 2000) return 'Reason is too long. Please shorten it.'
}
