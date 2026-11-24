import { useId, useRef } from 'react'
import { Textarea } from '@/shared/ui/components/textarea'
import { useAdmin } from './hooks'
import { tr } from './i18n'
import type { Locale } from './types'
import { Button } from './ui'

export function ArticleBodyField({ value, onChange, disabled, language }: {
  value: string
  onChange: (value: string) => void
  disabled: boolean
  language: Locale
}) {
  const { t } = useAdmin()
  const id = useId()
  const input = useRef<HTMLTextAreaElement>(null)

  function insertSubheading() {
    const field = input.current
    if (!field) return
    const text = field.value
    const caret = field.selectionStart
    const offset = caret === 0 || caret === text.length ? caret : text.lastIndexOf('\n', caret - 1) + 1
    const before = offset > 0 && text[offset - 1] !== '\n' ? '\n\n' : ''
    const after = offset < text.length ? '\n\n' : ''
    const placeholder = tr('subheading', language)
    const prefix = `${before}## `
    field.setRangeText(prefix + placeholder + after, offset, offset, 'end')
    field.focus()
    const start = offset + prefix.length
    field.setSelectionRange(start, start + placeholder.length)
    onChange(field.value)
  }

  return <div className='field'>
    <div className='article-body-field-head'>
      <label htmlFor={id}>{t('body')}</label>
      {!disabled && <Button type='button' variant='secondary' className='small' aria-controls={id} onClick={insertSubheading}>
        {t('addSubheading')}
      </Button>}
    </div>
    <Textarea id={id} ref={input} disabled={disabled} className='editor-body' value={value} onChange={event => onChange(event.target.value)}/>
  </div>
}
