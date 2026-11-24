import type { Article, StoryFollowUpInput } from '@creata/contracts'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { canPublishArticle } from './domain'
import { useAdmin, useStaff } from './hooks'
import { articleFollowUpKey, articleFollowUpPath, useStoryFollowUpMutation, type ArticleFollowUpResult } from './story-follow-up'
import { Button, ErrorPanel, Field, Loading } from './ui'

export function ArticleFollowUp({ article, disabled }: { article: Article; disabled: boolean }) {
  const { api, t } = useAdmin()
  const followUp = useQuery({
    queryKey: articleFollowUpKey(article.id),
    queryFn: () => api<ArticleFollowUpResult>(articleFollowUpPath(article.id)),
    staleTime: 0,
  })
  const staff = useStaff()
  const mutation = useStoryFollowUpMutation()
  const [changes, setChanges] = useState<Partial<StoryFollowUpInput>>({})

  const item = followUp.data?.item
  const canSchedule = followUp.data?.eligible
  const assignees = staff.data?.items.filter(person => canPublishArticle(person, article)) ?? []
  const dueDate = changes.dueDate ?? item?.dueDate ?? ''
  const assigneeId = changes.assigneeId ?? item?.assigneeId ?? ''
  const validAssignee = assignees.some(person => person.id === assigneeId)
  const changed = dueDate !== item?.dueDate || assigneeId !== item?.assigneeId
  const pending = disabled || mutation.isPending
  const canSave = canSchedule && !pending && !staff.error && !!dueDate && validAssignee && changed

  function submit(input: StoryFollowUpInput | null) {
    mutation.mutate({ articleId: article.id, input }, { onSuccess: () => setChanges({}) })
  }

  return <section className='article-follow-up' aria-labelledby='article-follow-up-heading'>
    <h3 id='article-follow-up-heading'>{t('storyFollowUps')}</h3>
    {followUp.isPending ? <Loading/> : followUp.error ? <ErrorPanel error={followUp.error} retry={() => void followUp.refetch()}/>
      : <form className='form-stack' onSubmit={event => {
        event.preventDefault()
        if (canSave) submit({ dueDate, assigneeId })
      }}>
        {canSchedule ? <div className='form-grid'>
          <Field label={t('followUpDueDate')}>
            <input type='date' value={dueDate} required disabled={pending}
              onChange={event => setChanges(previous => ({ ...previous, dueDate: event.target.value }))}/>
          </Field>
          <Field label={t('followUpAssignee')}>
            <select value={assigneeId} required disabled={pending || staff.isPending || !!staff.error}
              onChange={event => setChanges(previous => ({ ...previous, assigneeId: event.target.value }))}>
              <option value=''>{t('select')}</option>
              {item?.assigneeId && !assignees.some(person => person.id === item.assigneeId) &&
                <option value={item.assigneeId} disabled>{item.assigneeName || '—'}</option>}
              {assignees.map(person => <option key={person.id} value={person.id}>{person.name}</option>)}
            </select>
          </Field>
        </div> : <p className='callout'>{t('followUpNeedsTimeline')}</p>}
        {canSchedule && staff.error && <ErrorPanel error={staff.error} retry={() => void staff.refetch()}/>}
        {mutation.error && <ErrorPanel error={mutation.error}/>}
        {(canSchedule || item) && <div className='form-actions'>
          {item && <Button type='button' variant='secondary' disabled={pending} onClick={() => submit(null)}>
            {t('followUpComplete')}
          </Button>}
          {canSchedule && <Button type='submit' disabled={!canSave}>
            {t('followUpSave')}
          </Button>}
        </div>}
      </form>}
  </section>
}
