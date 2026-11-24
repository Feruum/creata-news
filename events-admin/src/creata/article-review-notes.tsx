import { ChevronDown, TriangleAlert } from 'lucide-react'
import { tr } from './i18n'
import type { Locale } from './types'

function isSingleSourceWarning(note: string) {
  const warning = 'Single source: editorial verification required'
  return note === warning || note.startsWith(`${warning} / `)
}

export function ArticleReviewNotes({ notes = [], locale, ecosystem = false }: {
  notes?: string[]
  locale: Locale
  ecosystem?: boolean
}) {
  const distinctNotes = [...new Set(notes.map(note => note.trim()).filter(Boolean))]
  const details = distinctNotes.filter(note => !isSingleSourceWarning(note))
  const title = tr('reviewNotes', locale)
  const warning = distinctNotes.length > details.length && <span className='review-notes-warning'>
    <TriangleAlert size={14} aria-hidden='true'/>{tr('reviewSingleSource', locale)}
  </span>

  return <section className='panel review-notes'>
    {details.length ? <details>
      <summary>
        <h2>
          <span className='review-notes-heading'>
            <span>{title} <small>{details.length}</small></span>
            <ChevronDown size={16} aria-hidden='true'/>
          </span>
          {warning}
        </h2>
      </summary>
      <ul className='review-notes-list' aria-label={title} tabIndex={0}>
        {details.map(note => <li key={note}>{note}</li>)}
      </ul>
    </details> : <>
      <h2>{title}</h2>
      {warning || <p className='panel-description'>{tr('noNotes', locale)}</p>}
    </>}
    {ecosystem && <p className='callout'>{tr('chiefHint', locale)}</p>}
  </section>
}
