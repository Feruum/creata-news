import { Plus, Trash2 } from 'lucide-react'
import type { SourceRef } from '@creata/contracts'
import { calendarDay } from './domain'
import { useAdmin } from './hooks'
import { Button, Field } from './ui'

export function ArticleSources({ sources, disabled, onChange }: {
  sources: SourceRef[]
  disabled: boolean
  onChange: (sources: SourceRef[]) => void
}) {
  const { t } = useAdmin()
  function update(index: number, patch: Partial<SourceRef>) {
    onChange(sources.map((source, position) => position === index ? { ...source, ...patch } : source))
  }

  return <section className='panel'>
    <h2>{t('facts')}</h2>
    {sources.map((source, index) => <div className='reference' key={index}>
      <div className='reference-head'>
        <span>{index + 1}</span>
        <button disabled={disabled} type='button' className='icon-button' aria-label={t('remove')}
          onClick={() => onChange(sources.filter((_, position) => position !== index))}><Trash2 size={13}/></button>
      </div>
      <Field label={t('sourceTitle')}>
        <input disabled={disabled} value={source.title} onChange={event => update(index, { title: event.target.value })}/>
      </Field>
      <Field label={t('sourceUrl')}>
        <input type='url' disabled={disabled} value={source.url} onChange={event => update(index, { url: event.target.value })}/>
      </Field>
      <Field label={t('sourceDate')}>
        <input type='date' disabled={disabled} value={calendarDay(source.publishedAt ?? '')} onChange={event => {
          const day = event.target.value
          update(index, { publishedAt: day ? `${day}T00:00:00Z` : undefined })
        }}/>
      </Field>
    </div>)}
    <Button type='button' variant='ghost' className='small' disabled={disabled} onClick={() => onChange([...sources, { title: '', url: '' }])}>
      <Plus size={13}/>{t('addReference')}
    </Button>
  </section>
}
