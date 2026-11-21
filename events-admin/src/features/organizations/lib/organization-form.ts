import { type Organization, OrganizationStatus } from '@studyverse/proto'
import { z } from 'zod'
import { EMPTY_PLATE_VALUE, getPlateTextLength, parsePlateJson } from '@/shared/lib/plate-utils'

const IMAGE_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/gif'])

export const organizationFormSchema = z.object({
  title: z
    .string()
    .trim()
    .min(2, 'Title must be at least 2 characters')
    .max(100, 'Title must be at most 100 characters'),
  description: z
    .string()
    .refine((value) => getPlateTextLength(parsePlateJson(value)) <= 2000, 'Description must be at most 2000 characters')
    .optional(),
  organizationTypeId: z.number().min(1, 'Organization type is required'),
  status: z.nativeEnum(OrganizationStatus),
  coverImage: z
    .instanceof(File)
    .refine((file) => file.size > 0 && file.size <= 5 * 1024 * 1024, 'Image must be between 1 byte and 5 MB')
    .refine((file) => IMAGE_TYPES.has(file.type), 'Choose a JPEG, PNG, WebP or GIF image')
    .optional(),
  removeImage: z.boolean(),
  website: z.string().url('Please enter a valid URL').optional().or(z.literal('')),
  instagram: z.string().optional(),
  telegramChannel: z.string().optional(),
  telegramChat: z.string().optional(),
  youtube: z.string().url('Please enter a valid URL').optional().or(z.literal('')),
  tiktok: z.string().optional(),
  linkedin: z.string().url('Please enter a valid URL').optional().or(z.literal(''))
})

export type OrganizationFormData = z.infer<typeof organizationFormSchema>

function comparableDescription(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(comparableDescription)
  if (value === null || typeof value !== 'object') return value
  const node = value as Record<string, unknown>
  return Object.fromEntries(
    Object.entries(node)
      // Plate assigns element IDs during normalization; those are not an edit.
      .filter(([key]) => key !== 'id' || !Array.isArray(node.children))
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, child]) => [key, comparableDescription(child)])
  )
}

export function hasOrganizationFormChanges(values: OrganizationFormData, initial: OrganizationFormData): boolean {
  for (const field of Object.keys(values) as (keyof OrganizationFormData)[]) {
    if (field !== 'description' && values[field] !== initial[field]) return true
  }
  return (
    JSON.stringify(comparableDescription(parsePlateJson(values.description))) !==
    JSON.stringify(comparableDescription(parsePlateJson(initial.description)))
  )
}

export function getOrganizationFormValues(organization?: Organization): OrganizationFormData {
  return {
    title: organization?.title ?? '',
    description: organization?.description ?? '',
    organizationTypeId: organization?.organizationTypeId ?? 0,
    status: organization?.status ?? OrganizationStatus.ACTIVE,
    removeImage: false,
    website: organization?.website ?? '',
    instagram: organization?.instagram ?? '',
    telegramChannel: organization?.telegramChannel ?? '',
    telegramChat: organization?.telegramChat ?? '',
    youtube: organization?.youtube ?? '',
    tiktok: organization?.tiktok ?? '',
    linkedin: organization?.linkedin ?? ''
  }
}

export function getOrganizationCreateInput(data: OrganizationFormData, uploadedImageUrl?: string) {
  const description = parsePlateJson(data.description)
  return {
    title: data.title,
    description: JSON.stringify(getPlateTextLength(description) === 0 ? EMPTY_PLATE_VALUE : description),
    organizationTypeId: data.organizationTypeId,
    status: data.status,
    imageUrl: uploadedImageUrl ?? (data.removeImage ? '' : undefined),
    website: data.website || '',
    instagram: data.instagram || '',
    telegramChannel: data.telegramChannel || '',
    telegramChat: data.telegramChat || '',
    youtube: data.youtube || '',
    tiktok: data.tiktok || '',
    linkedin: data.linkedin || ''
  }
}

export function getOrganizationUpdateInput(
  data: OrganizationFormData,
  organization: Pick<Organization, 'id' | 'title'>,
  uploadedImageUrl?: string
) {
  return {
    ...getOrganizationCreateInput(data, uploadedImageUrl),
    id: organization.id,
    title: data.title === organization.title ? undefined : data.title
  }
}
