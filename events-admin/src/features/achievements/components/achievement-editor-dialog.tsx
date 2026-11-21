import { useQuery } from '@connectrpc/connect-query'
import {
  type Achievement,
  AchievementEffect,
  AchievementRuleKind,
  AchievementStatus,
  listCategories,
  listEvents,
  listOrganizations
} from '@studyverse/proto'
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/shared/ui/components/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/shared/ui/components/dialog'
import { FileUpload } from '@/shared/ui/components/file-upload'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Textarea } from '@/shared/ui/components/textarea'
import { useAchievementClient, useRefreshAchievements } from '../api'
import { type AchievementFormValues, achievementFormSchema, validateAchievementImage } from '../validation'
import { AchievementCard } from './achievement-card'

export function AchievementEditorDialog({ achievement, onClose }: { achievement?: Achievement; onClose: () => void }) {
  const client = useAchievementClient()
  const refresh = useRefreshAchievements()
  const [values, setValues] = useState<AchievementFormValues>({
    title: achievement?.title || '',
    description: achievement?.description || '',
    imageUrl: achievement?.imageUrl || '',
    badgeIconUrl: achievement?.badgeIconUrl || '',
    backgroundColor: achievement?.backgroundColor || '#152034',
    accentColor: achievement?.accentColor || '#8bd7ff',
    effect: achievement?.effect ?? AchievementEffect.NONE,
    status: achievement?.status ?? AchievementStatus.DRAFT,
    ruleKind: achievement?.rule?.kind ?? AchievementRuleKind.UNSPECIFIED,
    threshold: achievement?.rule?.threshold || 1,
    eventId: achievement?.rule?.eventId || 0,
    organizationId: achievement?.rule?.organizationId || 0,
    categoryId: achievement?.rule?.categoryId || 0
  })
  const [pending, setPending] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const events = useQuery(listEvents, { page: 1, limit: 1000 })
  const organizations = useQuery(listOrganizations, { page: 1, limit: 1000 })
  const categories = useQuery(listCategories, { page: 1, limit: 1000 })
  const update = <K extends keyof AchievementFormValues>(key: K, value: AchievementFormValues[K]) =>
    setValues((previous) => ({ ...previous, [key]: value }))

  async function upload(file: File | undefined, field: 'imageUrl' | 'badgeIconUrl') {
    if (!file) return
    const message = validateAchievementImage(file)
    if (message) {
      setError(message)
      return
    }
    setUploading(true)
    setError('')
    try {
      const uploaded = await client.uploadAchievementImage({
        image: new Uint8Array(await file.arrayBuffer()),
        contentType: file.type,
        fileName: file.name,
        uploadKind: field === 'imageUrl' ? 'image' : 'badge'
      })
      update(field, uploaded.imageUrl)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Image upload failed')
    } finally {
      setUploading(false)
    }
  }

  async function uploadPreset(path: string, field: 'imageUrl' | 'badgeIconUrl') {
    setError('')
    setUploading(true)
    try {
      const response = await fetch(`${import.meta.env.BASE_URL}images/${path}`)
      if (!response.ok) throw new Error('Image could not be loaded')
      const image = await response.blob()
      await upload(new File([image], path.split('/').at(-1) || 'mascot.webp', { type: image.type }), field)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Image could not be uploaded')
    } finally {
      setUploading(false)
    }
  }

  async function save(event: React.FormEvent) {
    event.preventDefault()
    const parsed = achievementFormSchema.safeParse(values)
    if (!parsed.success) {
      setError(parsed.error.issues.map((issue) => issue.message).join('. '))
      return
    }
    const data = parsed.data
    setPending(true)
    setError('')
    try {
      await client.saveAchievement({
        achievement: {
          id: achievement?.id || 0,
          revision: achievement?.revision || 0,
          title: data.title,
          description: data.description,
          imageUrl: data.imageUrl,
          badgeIconUrl: data.badgeIconUrl,
          backgroundColor: data.backgroundColor,
          accentColor: data.accentColor,
          effect: data.effect,
          status: data.status,
          rule: data.ruleKind
            ? {
                kind: data.ruleKind,
                threshold: data.threshold,
                eventId: data.eventId || undefined,
                organizationId: data.organizationId || undefined,
                categoryId: data.categoryId || undefined
              }
            : undefined
        }
      })
      await refresh()
      toast.success('Achievement saved')
      onClose()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Could not save achievement')
    } finally {
      setPending(false)
    }
  }

  const selectClass = 'h-9 w-full rounded-md border bg-background px-3 text-sm'
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !pending && !uploading) onClose()
      }}
    >
      <DialogContent className='max-h-[92vh] overflow-y-auto sm:max-w-4xl'>
        <DialogHeader>
          <DialogTitle>{achievement ? 'Edit achievement' : 'Create achievement'}</DialogTitle>
          <DialogDescription>Design the reward and choose an optional automatic criterion.</DialogDescription>
        </DialogHeader>
        <form onSubmit={save} className='grid gap-8 md:grid-cols-[1fr_300px]'>
          <fieldset disabled={pending || uploading} className='min-w-0 space-y-5'>
            <div className='grid gap-2'>
              <Label htmlFor='achievement-title'>Title</Label>
              <Input
                id='achievement-title'
                required
                maxLength={200}
                value={values.title}
                onChange={(event) => update('title', event.target.value)}
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='achievement-description'>Description</Label>
              <Textarea
                id='achievement-description'
                maxLength={4000}
                rows={3}
                value={values.description}
                onChange={(event) => update('description', event.target.value)}
              />
            </div>
            {(['imageUrl', 'badgeIconUrl'] as const).map((field) => (
              <div key={field} className='space-y-2'>
                <Label htmlFor={`achievement-${field}`}>
                  {field === 'imageUrl' ? 'Card illustration (optional)' : 'Name badge icon (optional)'}
                </Label>
                <Input
                  id={`achievement-${field}`}
                  type='url'
                  placeholder='https://…'
                  value={values[field]}
                  onChange={(event) => update(field, event.target.value)}
                />
                <FileUpload
                  accept={{ 'image/png': ['.png'], 'image/jpeg': ['.jpg', '.jpeg'], 'image/webp': ['.webp'] }}
                  maxSize={5 * 1024 * 1024}
                  isUploading={uploading}
                  onChange={(files) => void upload(files[0], field)}
                  placeholder={`Upload ${field === 'imageUrl' ? 'illustration' : 'badge icon'} · PNG, JPEG, WebP · max 5 MB`}
                />
                {field === 'badgeIconUrl' && (
                  <div className='flex items-center gap-3'>
                    <Button
                      type='button'
                      variant='outline'
                      onClick={() => void uploadPreset('studyverse-badge.png', field)}
                    >
                      <img src={`${import.meta.env.BASE_URL}images/studyverse-badge.png`} alt='' className='size-5' />
                      Use Studyverse logo
                    </Button>
                    {values.badgeIconUrl && (
                      <img src={values.badgeIconUrl} alt='Selected name badge' className='size-8 object-contain' />
                    )}
                  </div>
                )}
                {field === 'imageUrl' && (
                  <fieldset className='flex flex-wrap gap-2' aria-label='Studyverse mascot variants'>
                    {[
                      ['welcome', 'Welcome'],
                      ['scholar', 'Scholar'],
                      ['star', 'Star']
                    ].map(([key, label]) => (
                      <Button
                        key={key}
                        type='button'
                        variant='outline'
                        className='h-auto flex-col gap-1 px-3 py-2'
                        onClick={() => void uploadPreset(`mascots/${key}.webp`, field)}
                      >
                        <img
                          src={`${import.meta.env.BASE_URL}images/mascots/${key}.webp`}
                          alt=''
                          className='size-14 object-contain'
                        />
                        {label}
                      </Button>
                    ))}
                  </fieldset>
                )}
                <p className='text-xs text-muted-foreground'>
                  {field === 'imageUrl'
                    ? 'Leave empty to use the Studyverse mascot.'
                    : 'Leave empty for no name badge. Only rewards with a badge icon can be selected next to a name.'}
                </p>
              </div>
            ))}
            <div className='grid grid-cols-2 gap-4'>
              {(['backgroundColor', 'accentColor'] as const).map((field) => (
                <div key={field} className='space-y-2'>
                  <Label htmlFor={`achievement-${field}`}>
                    {field === 'backgroundColor' ? 'Background color' : 'Accent color'}
                  </Label>
                  <Input
                    id={`achievement-${field}`}
                    type='color'
                    value={values[field]}
                    onChange={(event) => update(field, event.target.value)}
                  />
                </div>
              ))}
            </div>
            <div className='grid grid-cols-2 gap-4'>
              <div className='space-y-2'>
                <Label htmlFor='achievement-effect'>Effect</Label>
                <select
                  id='achievement-effect'
                  className={selectClass}
                  value={values.effect}
                  onChange={(event) => update('effect', Number(event.target.value))}
                >
                  <option value={AchievementEffect.NONE}>None</option>
                  <option value={AchievementEffect.HOLOGRAPHIC}>Holographic</option>
                  <option value={AchievementEffect.AURORA}>Aurora</option>
                  <option value={AchievementEffect.GOLD_FOIL}>Gold foil</option>
                  <option value={AchievementEffect.SPARKLE}>Sparkle</option>
                </select>
              </div>
              <div className='space-y-2'>
                <Label htmlFor='achievement-status'>Status</Label>
                <select
                  id='achievement-status'
                  className={selectClass}
                  value={values.status}
                  onChange={(event) => update('status', Number(event.target.value))}
                >
                  <option value={AchievementStatus.DRAFT}>Draft</option>
                  <option value={AchievementStatus.ACTIVE}>Active</option>
                  <option value={AchievementStatus.ARCHIVED}>Archived</option>
                </select>
              </div>
            </div>
            <div className='space-y-2'>
              <Label htmlFor='achievement-rule'>Automatic criterion</Label>
              <select
                id='achievement-rule'
                className={selectClass}
                value={values.ruleKind}
                onChange={(event) => {
                  const kind = Number(event.target.value)
                  setValues((previous) =>
                    kind === AchievementRuleKind.UNSPECIFIED
                      ? { ...previous, ruleKind: kind, threshold: 1, eventId: 0, organizationId: 0, categoryId: 0 }
                      : { ...previous, ruleKind: kind }
                  )
                }}
              >
                <option value={AchievementRuleKind.UNSPECIFIED}>Manual awards only</option>
                <option value={AchievementRuleKind.REGISTRATIONS}>Registrations</option>
                <option value={AchievementRuleKind.ATTENDANCE}>Attendance</option>
                <option value={AchievementRuleKind.SOC_GPA_ATTENDANCE}>SOC GPA attendance</option>
                <option value={AchievementRuleKind.IROS_ATTENDANCE}>IROS attendance</option>
              </select>
            </div>
            {values.ruleKind > 0 && (
              <div className='space-y-4 rounded-lg border p-4'>
                <div className='space-y-2'>
                  <Label htmlFor='achievement-threshold'>Required count</Label>
                  <Input
                    id='achievement-threshold'
                    type='number'
                    required
                    min={1}
                    max={100000}
                    step={1}
                    value={values.threshold}
                    onChange={(event) => update('threshold', event.target.valueAsNumber)}
                  />
                </div>
                <p className='text-xs text-muted-foreground'>
                  All selected filters must match the same event. Leave a filter unrestricted to include all.
                </p>
                {(
                  [
                    { key: 'eventId', title: 'Event', entries: events.data?.events },
                    { key: 'organizationId', title: 'Organization', entries: organizations.data?.organizations },
                    {
                      key: 'categoryId',
                      title: 'Category',
                      entries: categories.data?.categories.map((category) => ({
                        id: category.id,
                        title: category.name
                      }))
                    }
                  ] as const
                ).map(({ key, title, entries }) => (
                  <div key={key} className='space-y-2'>
                    <Label htmlFor={`achievement-${key}`}>{title}</Label>
                    <select
                      id={`achievement-${key}`}
                      className={selectClass}
                      value={values[key]}
                      onChange={(event) => update(key, Number(event.target.value))}
                    >
                      <option value={0}>Any {title.toLowerCase()}</option>
                      {values[key] > 0 && !entries?.some((entry) => entry.id === values[key]) && (
                        <option value={values[key]}>#{values[key]}</option>
                      )}
                      {entries?.map((entry) => (
                        <option key={entry.id} value={entry.id}>
                          {entry.title} (#{entry.id})
                        </option>
                      ))}
                    </select>
                    <Label htmlFor={`achievement-${key}-exact`}>{title} ID (optional)</Label>
                    <Input
                      id={`achievement-${key}-exact`}
                      type='number'
                      min={1}
                      max={2147483647}
                      step={1}
                      value={values[key] || ''}
                      placeholder={`Enter any ${title.toLowerCase()} ID`}
                      onChange={(event) => update(key, event.target.value === '' ? 0 : event.target.valueAsNumber)}
                    />
                    <p className='text-xs text-muted-foreground'>
                      Choose from loaded options or enter an exact ID, including historical records. Leave empty for any{' '}
                      {title.toLowerCase()}.
                    </p>
                  </div>
                ))}
                {(events.isError || organizations.isError || categories.isError) && (
                  <p role='alert' className='text-sm text-destructive'>
                    Some filter options could not be loaded.
                  </p>
                )}
              </div>
            )}
            <p className='text-xs text-muted-foreground'>
              Archiving stops new awards and preserves existing achievements.
            </p>
            {error && (
              <p role='alert' className='text-sm text-destructive'>
                {error}
              </p>
            )}
            <div className='flex gap-2'>
              <Button type='submit' disabled={pending || uploading}>
                {pending ? 'Saving…' : 'Save achievement'}
              </Button>
              <Button type='button' variant='outline' onClick={onClose}>
                Cancel
              </Button>
            </div>
          </fieldset>
          <aside className='space-y-4 md:sticky md:top-0 md:self-start'>
            <p className='text-sm font-medium'>Live preview</p>
            <AchievementCard achievement={values} />
            <p className='text-xs text-muted-foreground text-center'>
              Move your pointer over the card to preview its tilt.
            </p>
          </aside>
        </form>
      </DialogContent>
    </Dialog>
  )
}
