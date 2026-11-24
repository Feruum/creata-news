import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { LaunchSubmission, LaunchSubmissionReview, List, Locale } from '@creata/contracts'
import { useAdmin, useCategories } from './hooks'
import { localized } from './domain'
import { formatDate, locales } from './i18n'
import { Button, ErrorPanel, ExternalLink, Field, Loading, LocaleTabs, Modal } from './ui'

const queuePath = '/admin/launch-submissions'
const queueKey = ['launch-submissions']

function ReviewSubmission({ item, onClose }: { item: LaunchSubmission; onClose: () => void }) {
  const { api, t, toast, locale } = useAdmin()
  const client = useQueryClient()
  const categories = useCategories()
  const category = categories.data?.items.find(category => category.id === item.categoryId)
  const [language, setLanguage] = useState<Locale>(() =>
    item.description[locale].trim() ? locale : locales.find(lang => item.description[lang].trim()) ?? locale)
  const [note, setNote] = useState('')
  const mutation = useMutation({
    mutationFn: (action: LaunchSubmissionReview['action']) => api<LaunchSubmission>(
      `${queuePath}/${encodeURIComponent(item.id)}/review`,
      { method: 'POST', body: JSON.stringify({ action, expectedRevision: item.revision, note }) },
    ),
    onSuccess: (updated) => {
      void client.invalidateQueries({ queryKey: queueKey })
      void client.invalidateQueries({ queryKey: ['launches'] })
      toast(t(updated.status === 'published' ? 'launchReviewPublished' : 'launchReviewReturned'))
      onClose()
    },
  })

  return <Modal title={item.name} onClose={onClose} wide disablePointerDismissal>
    <div className="form-stack">
      <LocaleTabs value={language} onChange={setLanguage} />
      <dl className="form-stack">
        <div><dt><strong>{t('launchTagline')}</strong></dt><dd className="whitespace-pre-wrap break-words">{item.tagline[language] || '—'}</dd></div>
        <div><dt><strong>{t('description')}</strong></dt><dd className="whitespace-pre-wrap break-words">{item.description[language] || '—'}</dd></div>
        <div><dt><strong>{t('launchTeam')}</strong></dt><dd className="whitespace-pre-wrap break-words">{item.team[language] || '—'}</dd></div>
        <div className="form-grid">
          <div><dt><strong>{t('launchDate')}</strong></dt><dd>{formatDate(`${item.launchDate}T12:00:00Z`, locale)}</dd></div>
          {item.country && <div><dt><strong>{t('country')}</strong></dt><dd>{item.country}</dd></div>}
          {category && <div><dt><strong>{t('topic')}</strong></dt><dd>{localized(category.name, locale)}</dd></div>}
        </div>
      </dl>
      <div className="form-actions">
        <ExternalLink href={item.url}>{t('launchWebsite')}</ExternalLink>
        {item.photo && <ExternalLink href={item.photo}>{t('launchPhoto')}</ExternalLink>}
      </div>
      <Field label={t('launchReviewNote')}>
        <textarea rows={3} maxLength={2000} value={note} onChange={event => setNote(event.target.value)} />
      </Field>
      {mutation.error && <ErrorPanel error={mutation.error} retry={() => { onClose(); void client.invalidateQueries({ queryKey: queueKey }) }} />}
      <div className="form-actions">
        <Button type="button" variant="secondary" disabled={!note.trim() || mutation.isPending} onClick={() => mutation.mutate('request_changes')}>
          {t('launchReviewReturn')}
        </Button>
        <Button type="button" disabled={mutation.isPending} onClick={() => mutation.mutate('publish')}>
          {t('publish')}
        </Button>
      </div>
    </div>
  </Modal>
}

export function LaunchReviewQueue() {
  const { api, t } = useAdmin()
  const [reviewing, setReviewing] = useState<LaunchSubmission>()
  const queue = useQuery({ queryKey: queueKey, queryFn: () => api<List<LaunchSubmission>>(queuePath) })

  return <section className="panel form-stack" aria-labelledby="launch-review-heading">
    <h2 id="launch-review-heading">{t('launchReviewQueue')}</h2>
    {queue.isPending ? <Loading /> : queue.error ? <ErrorPanel error={queue.error} retry={() => void queue.refetch()} />
      : !queue.data.items.length ? <p>{t('launchReviewEmpty')}</p>
        : <ul className="form-stack">
          {queue.data.items.map(item => <li className="form-actions" key={item.id}>
            <strong>{item.name}</strong>
            <Button type="button" variant="secondary" onClick={() => setReviewing(item)}
              aria-label={`${t('launchReviewOpen')} · ${item.name}`}>{t('launchReviewOpen')}</Button>
          </li>)}
        </ul>}
    {reviewing && <ReviewSubmission key={reviewing.id} item={reviewing} onClose={() => setReviewing(undefined)} />}
  </section>
}
