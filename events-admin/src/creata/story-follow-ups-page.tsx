import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { canAccessStoryFollowUps, localized, storyFollowUpStatus } from './domain'
import { useAdmin } from './hooks'
import { formatDate } from './i18n'
import { storyFollowUpsKey, storyFollowUpsPath, useStoryFollowUpMutation, type StoryFollowUpQueue } from './story-follow-up'
import { Badge, Button, Empty, ErrorPanel, Loading, PageHead } from './ui'

export function StoryFollowUpsPage() {
  const { api, locale, session, t } = useAdmin()
  const allowed = canAccessStoryFollowUps(session?.user)
  const queue = useQuery({
    queryKey: storyFollowUpsKey,
    queryFn: () => api<StoryFollowUpQueue>(storyFollowUpsPath),
    enabled: allowed,
  })
  const mutation = useStoryFollowUpMutation()
  const now = new Date().toISOString()

  return <>
    <PageHead title={t('storyFollowUps')}/>
    {mutation.error && <ErrorPanel error={mutation.error}/>}
    {!allowed ? <p className='callout'>{t('forbidden')}</p>
      : queue.isPending ? <Loading/> : queue.error ? <ErrorPanel error={queue.error} retry={() => void queue.refetch()}/>
        : !queue.data.items.length ? <div className='table-panel'><Empty title={t('followUpEmpty')}/></div>
          : <ol className='story-follow-up-queue panel' role='list'>
            {queue.data.items.map(item => <li key={item.articleId}>
              <div className='story-follow-up-story'>
                <Link to='/articles/$articleId' params={{ articleId: item.articleId }} search={{ tab: 'story' }} className='row-title'>
                  {localized(item.title, locale) || t('untitled')}
                </Link>
                <span className='muted'>{item.assigneeName || '—'}</span>
              </div>
              <div className='story-follow-up-date'>
                <time dateTime={item.dueDate}>{formatDate(item.dueDate, locale, false, true)}</time>
                <Badge status={storyFollowUpStatus(item.dueDate, now)}/>
              </div>
              <Button type='button' variant='secondary' className='small' disabled={mutation.isPending}
                onClick={() => mutation.mutate({ articleId: item.articleId, input: null })}>
                {t('followUpComplete')}
              </Button>
            </li>)}
          </ol>}
  </>
}
