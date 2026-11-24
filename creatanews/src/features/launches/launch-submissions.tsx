'use client'

import Link from 'next/link'
import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { ApiError, type Category, type LaunchSubmission, type LaunchSubmissionCreate, type List, type Locale } from '@creata/contracts'
import { useAuth } from '@/components/auth-context'
import { copy, localePath, locales, localize } from '@/lib/i18n'
import { submissionCopy } from './copy'
import styles from './launches.module.css'

const endpoint = '/account/launch-submissions'
const textFields = ['tagline', 'description', 'team'] as const
const emptyText = { ru: '', kk: '', en: '' }

function SubmissionForm({ item, locale, categories, busy, onSave, onClose }: {
  item: LaunchSubmission | null
  locale: Locale
  categories: Category[]
  busy: boolean
  onSave: (fields: LaunchSubmissionCreate) => void
  onClose: () => void
}) {
  const t = submissionCopy[locale], common = copy[locale]
  const [language, setLanguage] = useState(locale)
  const [fields, setFields] = useState<LaunchSubmissionCreate>(() => item ?? { categoryId: 'startups' })
  const update = <Key extends keyof LaunchSubmissionCreate>(key: Key, value: LaunchSubmissionCreate[Key]) => {
    setFields(current => ({ ...current, [key]: value }))
  }
  function save(event: FormEvent) {
    event.preventDefault()
    const { name, tagline, description, team, url, launchDate, country, categoryId, photo } = fields
    onSave({ name: name?.trim(), tagline, description, team, url: url?.trim(), launchDate, country: country?.trim(), categoryId, photo: photo?.trim() })
  }
  return <form className="profile-form" onSubmit={save}>
    <h3>{item ? t.edit : t.add}</h3>
    <div className="form-grid">
      <label className="field">{t.name}<input value={fields.name ?? ''} maxLength={200} onChange={event => update('name', event.target.value)} /></label>
      <label className="field">{t.website}<input type="url" value={fields.url ?? ''} onChange={event => update('url', event.target.value)} /></label>
      <label className="field">{t.date}<input type="date" value={fields.launchDate ?? ''} onChange={event => update('launchDate', event.target.value)} /></label>
      <label className="field">{common.country}<input value={fields.country ?? ''} maxLength={100} onChange={event => update('country', event.target.value)} /></label>
      <label className="field">{common.topics}<select value={fields.categoryId ?? ''} onChange={event => update('categoryId', event.target.value)}>
        <option value="">{common.allTopics}</option>
        {categories.map(category => <option key={category.id} value={category.id}>{localize(category.name, locale)}</option>)}
      </select></label>
      <label className="field">{t.photo}<input type="url" value={fields.photo ?? ''} onChange={event => update('photo', event.target.value)} /></label>
    </div>
    <label className="field">{common.language}<select value={language} onChange={event => setLanguage(event.target.value as Locale)}>
      <option value="ru">Русский</option><option value="kk">Қазақша</option><option value="en">English</option>
    </select></label>
    <p>{t.translationHint}</p>
    {textFields.map(key => <label key={key} className="field">{t[key]}
      <textarea rows={key === 'tagline' ? 2 : 4} maxLength={20000} value={fields[key]?.[language] ?? ''}
        onChange={event => update(key, { ...(fields[key] ?? emptyText), [language]: event.target.value })} />
    </label>)}
    <div className="form-actions">
      <button className="button button-accent" disabled={busy}>{busy ? common.saving : t.save}</button>
      <button type="button" className="button button-outline" onClick={onClose} disabled={busy}>{common.close}</button>
    </div>
  </form>
}

export function LaunchSubmissions({ locale, categories }: { locale: Locale; categories: Category[] }) {
  const { request } = useAuth()
  const t = submissionCopy[locale], common = copy[locale]
  const [items, setItems] = useState<LaunchSubmission[]>()
  const [editing, setEditing] = useState<LaunchSubmission | null | undefined>()
  const [busy, setBusy] = useState(false)
  const [loadFailed, setLoadFailed] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const load = useCallback(async () => {
    setLoadFailed(false)
    try {
      const { items } = await request<List<LaunchSubmission>>(endpoint)
      setItems(items)
    } catch {
      setLoadFailed(true)
    }
  }, [request])
  useEffect(() => { void load() }, [load])

  async function mutate(action: 'save' | 'submit', item: LaunchSubmission | null, fields?: LaunchSubmissionCreate) {
    setError('')
    setMessage('')
    if (action === 'submit' && item && (!item.name.trim() || !item.url || !item.launchDate
      || textFields.some(field => !locales.some(lang => item[field][lang].trim())))) {
      setError(t.incomplete)
      return
    }
    setBusy(true)
    const path = item ? `${endpoint}/${encodeURIComponent(item.id)}` : endpoint
    try {
      const next = await request<LaunchSubmission>(action === 'submit' ? `${path}/submit` : path, {
        method: action === 'save' && item ? 'PATCH' : 'POST',
        body: JSON.stringify({ ...fields, ...(item ? { expectedRevision: item.revision } : {}) }),
      })
      setItems(previous => {
        const items = previous ?? []
        return items.some(current => current.id === next.id)
          ? items.map(current => current.id === next.id ? next : current)
          : [next, ...items]
      })
      setEditing(undefined)
      setMessage(action === 'submit' ? t.submitted : common.savedChanges)
    } catch (error) {
      setError(error instanceof ApiError && error.status === 409 ? t.conflict : common.operationFailed)
    } finally {
      setBusy(false)
    }
  }

  return <section id="launch-submissions" className={styles.submissions} aria-labelledby="launch-submissions-title">
    <div className={styles.submissionHeading}>
      <h2 id="launch-submissions-title">{t.title}</h2>
      {editing === undefined && <button className="button button-accent" onClick={() => { setEditing(null); setError(''); setMessage('') }} disabled={!items || busy}>{t.add}</button>}
    </div>
    {editing !== undefined && <SubmissionForm key={editing?.id ?? 'new'} item={editing} locale={locale} categories={categories} busy={busy}
      onSave={fields => void mutate('save', editing, fields)} onClose={() => { setEditing(undefined); setError('') }} />}
    {error && <div className="form-actions"><p role="alert" className="inline-message error">{error}</p>
      {error === t.conflict && <button className="button button-outline" onClick={() => { setEditing(undefined); setError(''); void load() }}>{t.retry}</button>}
    </div>}
    {message && <p role="status" className="inline-message">{message}</p>}
    {loadFailed ? <div className="form-actions"><p role="alert">{common.operationFailed}</p><button className="button button-outline" onClick={() => void load()}>{t.retry}</button></div>
      : !items ? <p role="status">{common.loading}</p>
        : !items.length && editing === undefined ? <p>{t.empty}</p> : null}
    {items?.map(item => <article className={styles.submission} key={item.id}>
      <div className={styles.submissionHeading}><h3>{item.name || t.unnamed}</h3><span>{t[item.status]}</span></div>
      {item.reviewNote && <div><strong>{t.reviewNote}</strong><p className={styles.prose}>{item.reviewNote}</p></div>}
      {(item.status === 'draft' || item.status === 'changes_requested') && <div className="form-actions">
        <button type="button" className="button button-outline" disabled={busy || editing !== undefined} onClick={() => { setEditing(item); setError(''); setMessage('') }}>{t.edit}</button>
        <button type="button" className="button button-accent" disabled={busy || editing !== undefined} onClick={() => void mutate('submit', item)}>{t.submit}</button>
      </div>}
      {item.launchSlug && <Link className="button button-outline" href={localePath(locale, `/launches/${item.launchSlug}`)}>{t.open}</Link>}
    </article>)}
  </section>
}
