import { createClient } from '@connectrpc/connect'
import { useTransport } from '@connectrpc/connect-query'
import { zodResolver } from '@hookform/resolvers/zod'
import { type Organization, OrganizationStatus, OrganizationsService } from '@studyverse/proto'
import { Loader2 } from 'lucide-react'
import type { Value } from 'platejs'
import { Suspense, useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { PlateEditor } from '@/features/events'
import { clearServerError, getErrorMessage, handleServerErrors } from '@/shared/lib/form-utils'
import { getPlateTextLength, parsePlateJson } from '@/shared/lib/plate-utils'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { ImageUpload } from '@/shared/ui/components/file-upload'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/shared/ui/components/form'
import { Input } from '@/shared/ui/components/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { useCreateOrganization, useUpdateOrganization } from '../api/mutations'
import { useOrganizationTypes } from '../api/queries'
import {
  getOrganizationCreateInput,
  getOrganizationFormValues,
  getOrganizationUpdateInput,
  hasOrganizationFormChanges,
  type OrganizationFormData,
  organizationFormSchema
} from '../lib/organization-form'

interface OrganizationFormContentProps {
  organization?: Organization
  onSaved: () => void
  onCancel: () => void
  onDirtyChange?: (dirty: boolean) => void
  onSubmittingChange?: (pending: boolean) => void
}

interface OrganizationEditorFormProps extends Omit<OrganizationFormContentProps, 'organization'> {
  organizationId?: number
  initialValues?: Partial<OrganizationFormData>
  existingImageUrl?: string
}

const SOCIAL_FIELDS = [
  { name: 'website', label: 'Website', placeholder: 'https://example.com' },
  { name: 'instagram', label: 'Instagram', placeholder: 'https://instagram.com/...' },
  { name: 'telegramChannel', label: 'Telegram Channel', placeholder: 'https://t.me/...' },
  { name: 'telegramChat', label: 'Telegram Chat', placeholder: 'https://t.me/...' },
  { name: 'youtube', label: 'YouTube', placeholder: 'https://youtube.com/...' },
  { name: 'tiktok', label: 'TikTok', placeholder: 'https://tiktok.com/@...' },
  { name: 'linkedin', label: 'LinkedIn', placeholder: 'https://linkedin.com/...' }
] as const

export function OrganizationFormContent({ organization, ...props }: OrganizationFormContentProps) {
  return (
    <OrganizationEditorForm
      {...props}
      organizationId={organization?.id}
      initialValues={getOrganizationFormValues(organization)}
      existingImageUrl={organization?.imageUrl}
    />
  )
}

function OrganizationEditorForm({
  organizationId,
  initialValues,
  existingImageUrl,
  onSaved,
  onCancel,
  onDirtyChange,
  onSubmittingChange
}: OrganizationEditorFormProps) {
  const transport = useTransport()
  const organizationsClient = useMemo(() => createClient(OrganizationsService, transport), [transport])
  const [startingValues] = useState(() => ({ ...getOrganizationFormValues(), ...initialValues }))
  const form = useForm<OrganizationFormData>({
    resolver: zodResolver(organizationFormSchema),
    defaultValues: startingValues,
    mode: 'onBlur'
  })
  const organizationTypes = useOrganizationTypes({ page: 1, limit: 1000 }).data.organizationTypes
  const createMutation = useCreateOrganization()
  const updateMutation = useUpdateOrganization()
  const removeImage = form.watch('removeImage')
  const coverImage = form.watch('coverImage')
  const { isSubmitting: isFormSubmitting } = form.formState
  const hasUnsavedChanges = hasOrganizationFormChanges(form.watch(), startingValues)
  const isSubmitting = createMutation.isPending || updateMutation.isPending || isFormSubmitting
  const isEditMode = organizationId !== undefined

  useEffect(() => onDirtyChange?.(hasUnsavedChanges), [hasUnsavedChanges, onDirtyChange])
  useEffect(() => onSubmittingChange?.(isSubmitting), [isSubmitting, onSubmittingChange])
  useEffect(
    () => () => {
      onDirtyChange?.(false)
      onSubmittingChange?.(false)
    },
    [onDirtyChange, onSubmittingChange]
  )

  const handleSubmit = form.handleSubmit(async (data) => {
    form.clearErrors('root')
    try {
      let uploadedImageUrl: string | undefined
      if (data.coverImage) {
        const uploaded = await organizationsClient.uploadOrganizationImage({
          organizationId: organizationId ?? 0,
          image: new Uint8Array(await data.coverImage.arrayBuffer()),
          contentType: data.coverImage.type
        })
        uploadedImageUrl = uploaded.publicUrl
      }

      if (organizationId !== undefined) {
        await updateMutation.mutateAsync(
          getOrganizationUpdateInput(data, { id: organizationId, title: startingValues.title }, uploadedImageUrl)
        )
      } else {
        await createMutation.mutateAsync(getOrganizationCreateInput(data, uploadedImageUrl))
      }
      form.reset({ ...data, coverImage: undefined, removeImage: false })
      onDirtyChange?.(false)
      toast.success(isEditMode ? 'Organization updated!' : 'Organization created!', { position: 'top-center' })
      onSaved()
    } catch (error) {
      form.setError('root', { message: getErrorMessage(error) })
      handleServerErrors(form, error, { showToast: false })
    }
  })

  return (
    <Form {...form}>
      <form onSubmit={handleSubmit} className='organization-form-content @container flex min-h-0 flex-1 flex-col'>
        <div className='min-h-0 flex-1 overflow-y-auto overscroll-contain'>
          <fieldset disabled={isSubmitting} className='grid min-w-0 gap-6 p-5 @min-[640px]:grid-cols-2'>
            <legend className='sr-only'>Organization settings</legend>
            <div className='min-w-0 space-y-5'>
              <div className='space-y-4'>
                <h3 className='text-sm font-medium'>Basic Information</h3>
                <FormField
                  control={form.control}
                  name='title'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Name</FormLabel>
                      <FormControl>
                        <Input
                          placeholder='Student Council...'
                          {...field}
                          onChange={(event) => {
                            field.onChange(event)
                            clearServerError(form, 'title')
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='description'
                  render={({ field }) => {
                    const plateValue = parsePlateJson(field.value)
                    return (
                      <FormItem>
                        <FormLabel>Description</FormLabel>
                        <FormControl>
                          <PlateEditor
                            value={plateValue}
                            readOnly={isSubmitting}
                            onChange={(value: Value) => {
                              if (isSubmitting) return
                              field.onChange(JSON.stringify(value))
                              clearServerError(form, 'description')
                            }}
                            placeholder='What does this organization do?'
                            minHeight='120px'
                          />
                        </FormControl>
                        <p className='text-right text-xs text-muted-foreground'>
                          {getPlateTextLength(plateValue)}/2000
                        </p>
                        <FormMessage />
                      </FormItem>
                    )
                  }}
                />
                <div className='grid grid-cols-2 gap-3'>
                  <FormField
                    control={form.control}
                    name='organizationTypeId'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Type</FormLabel>
                        <Select
                          disabled={isSubmitting}
                          value={field.value ? String(field.value) : ''}
                          onValueChange={(value) => {
                            if (value !== null) field.onChange(Number(value))
                            clearServerError(form, 'organizationTypeId')
                          }}
                        >
                          <FormControl>
                            <SelectTrigger className='w-full'>
                              <SelectValue className='block! min-w-0 truncate' placeholder='Select type...' />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {organizationTypes.map((type) => (
                              <SelectItem key={type.id} value={String(type.id)}>
                                {type.title}
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
                    name='status'
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Status</FormLabel>
                        <Select
                          disabled={isSubmitting}
                          value={String(field.value)}
                          onValueChange={(value) => {
                            if (value !== null) field.onChange(Number(value))
                            clearServerError(form, 'status')
                          }}
                        >
                          <FormControl>
                            <SelectTrigger className='w-full'>
                              <SelectValue placeholder='Select status...' />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value={String(OrganizationStatus.ACTIVE)}>Active</SelectItem>
                            <SelectItem value={String(OrganizationStatus.ARCHIVED)}>Archived</SelectItem>
                            <SelectItem value={String(OrganizationStatus.FROZEN)}>Frozen</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>
              <div className='space-y-3 border-t pt-4'>
                <h3 className='text-sm font-medium'>Organization Image</h3>
                <p className='text-xs text-muted-foreground'>JPEG, PNG, WebP or GIF, up to 5 MB.</p>
                <FormField
                  control={form.control}
                  name='coverImage'
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <ImageUpload
                          value={field.value ?? (!removeImage ? existingImageUrl : undefined)}
                          disabled={isSubmitting}
                          className='max-w-48'
                          aspectRatio='square'
                          showRemoveButton={!existingImageUrl || !!coverImage}
                          onChange={(file) => {
                            field.onChange(file ?? undefined)
                            form.setValue('removeImage', !file && !!existingImageUrl, { shouldDirty: true })
                            clearServerError(form, 'coverImage')
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                {existingImageUrl && !removeImage && !coverImage && (
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    disabled={isSubmitting}
                    onClick={() => form.setValue('removeImage', true, { shouldDirty: true })}
                  >
                    Remove image
                  </Button>
                )}
                {existingImageUrl && removeImage && !coverImage && (
                  <div className='flex items-center gap-3'>
                    <p className='text-xs text-muted-foreground'>The current image will be removed when you save.</p>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      disabled={isSubmitting}
                      onClick={() => form.setValue('removeImage', false, { shouldDirty: true })}
                    >
                      Undo
                    </Button>
                  </div>
                )}
              </div>
            </div>
            <div className='min-w-0 space-y-4'>
              <h3 className='text-sm font-medium'>Social Links</h3>
              <p className='text-xs text-muted-foreground'>Optional social profiles and website.</p>
              {SOCIAL_FIELDS.map(({ name, label, placeholder }) => (
                <FormField
                  key={name}
                  control={form.control}
                  name={name}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{label}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={placeholder}
                          onChange={(event) => {
                            field.onChange(event)
                            clearServerError(form, name)
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ))}
            </div>
          </fieldset>
        </div>
        <div className='shrink-0 space-y-3 border-t px-5 py-4'>
          {form.formState.errors.root?.message && (
            <p role='alert' className='text-sm text-destructive'>
              {form.formState.errors.root.message}
            </p>
          )}
          <div className='flex justify-end gap-2'>
            <Button type='button' variant='outline' onClick={onCancel} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type='submit' disabled={isSubmitting || !hasUnsavedChanges}>
              {isSubmitting && <Loader2 className='mr-2 size-4 animate-spin' />}
              {isEditMode ? 'Save Changes' : 'Create Organization'}
            </Button>
          </div>
        </div>
      </form>
    </Form>
  )
}

interface CreateOrganizationFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  organizationId?: number
  initialValues?: Partial<OrganizationFormData>
  existingImageUrl?: string
}

export function CreateOrganizationForm({ open, onOpenChange, ...props }: CreateOrganizationFormProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className='flex max-h-[90vh] w-[min(800px,calc(100vw-2rem))] flex-col gap-0 overflow-hidden p-0'
        style={{ overscrollBehavior: 'contain' }}
        showCloseButton={false}
      >
        <DialogHeader className='shrink-0 border-b px-5 py-4'>
          <DialogTitle>{props.organizationId !== undefined ? 'Edit Organization' : 'Create Organization'}</DialogTitle>
        </DialogHeader>
        {open && (
          <OrganizationEditorForm {...props} onSaved={() => onOpenChange(false)} onCancel={() => onOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  )
}

export function CreateOrganizationTrigger({ children }: { children: React.ReactNode }) {
  const [isOpen, setIsOpen] = useState(false)
  return (
    <>
      <button type='button' onClick={() => setIsOpen(true)} className='text-left'>
        {children}
      </button>
      <Suspense fallback={null}>
        <CreateOrganizationForm open={isOpen} onOpenChange={setIsOpen} />
      </Suspense>
    </>
  )
}
