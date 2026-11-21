import { useMutation } from '@tanstack/react-query'
import { ImagePlus } from 'lucide-react'
import type { Media } from '@creata/contracts'
import { useAdmin } from './hooks'
import { Button, ErrorPanel, Field } from './ui'
import { newspaperText } from './newspaper-copy'

export function NewspaperPhoto({ value, onChange, onBusy, disabled = false }: { value: Media; onChange: (media: Media) => void; onBusy: (busy: boolean) => void; disabled?: boolean }) {
  const { api, t, locale } = useAdmin()
  const upload = useMutation({ mutationFn: async (file: File) => {
    onBusy(true)
    try {
      const form = new FormData(); form.set('file', file); form.set('credit', value.credit); form.set('rights', value.rights); form.set('caption', value.caption)
      return await api<Media>('/admin/media', { method: 'POST', body: form })
    } finally { onBusy(false) }
  }, onSuccess: media => onChange(media) })
  const busy = disabled || upload.isPending
  return <div className="newspaper-photo form-stack">
    {value.url ? <img src={value.url} alt={value.caption} className="newspaper-photo-preview"/> : <div className="cover-placeholder"><ImagePlus size={26}/></div>}
    <Field label={t('credit')}><input value={value.credit} disabled={busy} onChange={e => onChange({ ...value, credit: e.target.value })}/></Field>
    <Field label={t('rights')} hint={t('rightsHint')}><input value={value.rights} disabled={busy} onChange={e => onChange({ ...value, rights: e.target.value })}/></Field>
    <Field label={t('caption')}><input value={value.caption} disabled={busy} onChange={e => onChange({ ...value, caption: e.target.value })}/></Field>
    <Field label={t('upload')} hint={newspaperText('photosHelp', locale)}><input type="file" accept="image/jpeg,image/png,image/webp,image/gif" disabled={busy || !value.credit.trim() || !value.rights.trim()} onChange={e => { const file = e.target.files?.[0]; if (file) upload.mutate(file); e.target.value = '' }}/></Field>
    {value.url && <Button type="button" variant="ghost" disabled={busy} onClick={() => onChange({ ...value, url: '' })}>{t('remove')}</Button>}
    {upload.isPending && <p role="status">{t('saving')}</p>}{upload.error && <ErrorPanel error={upload.error}/>}
  </div>
}
