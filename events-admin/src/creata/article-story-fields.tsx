import { Plus, Trash2 } from 'lucide-react'
import type { ActionGuide, Locale, StoryUpdate } from '@creata/contracts'
import { emptyText, storyUpdateDateInput, storyUpdateTimestamp } from './domain'
import { useAdmin } from './hooks'
import { Button, Field } from './ui'

export function ArticleStoryFields({ guide, updates, language, disabled, onGuideChange, onUpdatesChange }: {
  guide?: ActionGuide
  updates: StoryUpdate[]
  language: Locale
  disabled: boolean
  onGuideChange: (guide: ActionGuide | undefined) => void
  onUpdatesChange: (updates: StoryUpdate[]) => void
}) {
  const { t } = useAdmin()

  return <div className='form-stack article-story-fields'>
    {(guide || updates.length > 0) && <p className='callout'>{t('storyApprovalHint')}</p>}
    <section className='article-story-section'>
      <h3>{t('storyGuide')}</h3>
      {guide ? <div className='form-stack'>
        <Field label={t('storyGuideAudience')}>
          <textarea disabled={disabled} lang={language} rows={2} value={guide.audience[language]} onChange={event =>
            onGuideChange({ ...guide, audience: { ...guide.audience, [language]: event.target.value } })}/>
        </Field>
        <Field label={t('storyGuideSteps')}>
          <textarea disabled={disabled} lang={language} rows={3} value={guide.steps[language]} onChange={event =>
            onGuideChange({ ...guide, steps: { ...guide.steps, [language]: event.target.value } })}/>
        </Field>
        <Field label={t('storyGuideSource')}>
          <input type='url' disabled={disabled} value={guide.sourceUrl} onChange={event =>
            onGuideChange({ ...guide, sourceUrl: event.target.value })}/>
        </Field>
        <Button type='button' variant='ghost' disabled={disabled} onClick={() => onGuideChange(undefined)}>
          <Trash2 size={14}/>{t('storyGuideRemove')}
        </Button>
      </div> : <Button type='button' variant='secondary' disabled={disabled} onClick={() =>
        onGuideChange({ audience: emptyText(), steps: emptyText(), sourceUrl: '' })}>
        <Plus size={14}/>{t('storyGuideAdd')}
      </Button>}
    </section>
    <section className='article-story-section'>
      <h3>{t('storyUpdates')}</h3>
      <div className='form-stack'>
        {updates.map((update, index) => {
          const change = (patch: Partial<StoryUpdate>) =>
            onUpdatesChange(updates.map(item => item.id === update.id ? { ...item, ...patch } : item))

          return <fieldset className='article-story-update form-stack' key={update.id}>
            <legend>{t('storyUpdate')} {index + 1}</legend>
            <Field label={t('storyUpdateDate')}>
              <input type='datetime-local' step='any' required disabled={disabled} value={storyUpdateDateInput(update.occurredAt)}
                onChange={event => change({ occurredAt: storyUpdateTimestamp(event.target.value) })}/>
            </Field>
            <Field label={t('storyUpdateTitle')}>
              <input disabled={disabled} lang={language} value={update.title[language]} onChange={event =>
                change({ title: { ...update.title, [language]: event.target.value } })}/>
            </Field>
            <Field label={t('storyUpdateBody')}>
              <textarea disabled={disabled} lang={language} value={update.body[language]} onChange={event =>
                change({ body: { ...update.body, [language]: event.target.value } })}/>
            </Field>
            <Field label={t('storyUpdateSource')}>
              <input type='url' disabled={disabled} value={update.sourceUrl} onChange={event => change({ sourceUrl: event.target.value })}/>
            </Field>
            <Field label={t('storyUpdateArticle')} hint={t('storyUpdateArticleHint')}>
              <input disabled={disabled} value={update.articleSlug ?? ''} spellCheck={false}
                onChange={event => change({ articleSlug: event.target.value || undefined })}/>
            </Field>
            <Button type='button' variant='ghost' disabled={disabled}
              onClick={() => onUpdatesChange(updates.filter(item => item.id !== update.id))}>
              <Trash2 size={14}/>{t('storyUpdateRemove')}
            </Button>
          </fieldset>
        })}
        <Button type='button' variant='secondary' disabled={disabled} onClick={() => onUpdatesChange([...updates, {
          id: crypto.randomUUID(), occurredAt: '', title: emptyText(), body: emptyText(), sourceUrl: '',
        }])}>
          <Plus size={14}/>{t('storyUpdateAdd')}
        </Button>
      </div>
    </section>
  </div>
}
