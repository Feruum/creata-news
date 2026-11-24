import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import type { Locale } from '@/lib/models'
import { localePath } from '@/lib/i18n'
import { formatStoryTime, type localizedActionGuide, type localizedStoryUpdates } from '@/lib/story-content'
import { storyCopy } from '@/lib/story-copy'
import { FollowStoryButton } from './followed-stories'
import styles from './article-view.module.css'

export function StoryTimeline({ updates, articleId, slug, locale, preview }: {
  updates: ReturnType<typeof localizedStoryUpdates>
  articleId: string
  slug: string
  locale: Locale
  preview: boolean
}) {
  const t = storyCopy[locale]
  return (
    <section id="article-timeline" className={styles.timeline} aria-labelledby="article-timeline-title">
      <div className={styles.timelineHeader}>
        <h2 id="article-timeline-title">{t.timeline}</h2>
        {!preview && <FollowStoryButton key={articleId} id={articleId} slug={slug} locale={locale}
          updateIds={updates.map(update => update.id)} />}
      </div>
      <ol className={styles.timelineList}>
        {updates.map((update, index) => (
          <li key={update.id} id={`story-update-${update.id}`} className={styles.timelineItem}>
            <div className={styles.updateMeta}>
              {index === 0 && <strong>{t.latest}</strong>}
              <time dateTime={update.occurredAt}>{formatStoryTime(update.occurredAt, locale)}</time>
            </div>
            <h3>{update.title}</h3>
            <p>{update.body}</p>
            <div className={styles.storyLinks}>
              <a href={update.sourceUrl} target="_blank" rel="noopener noreferrer">
                {t.source} <ArrowUpRight size={14} aria-hidden="true" />
              </a>
              {update.articleSlug && update.articleSlug !== slug && (
                <Link href={localePath(locale, `/articles/${encodeURIComponent(update.articleSlug)}`)}>{t.related}</Link>
              )}
            </div>
          </li>
        ))}
      </ol>
    </section>
  )
}

export function ArticleActionGuide({ guide, locale }: {
  guide: NonNullable<ReturnType<typeof localizedActionGuide>>
  locale: Locale
}) {
  const t = storyCopy[locale]
  return (
    <section id="article-guide" className={styles.guide} aria-labelledby="article-guide-title">
      <h2 id="article-guide-title">{t.guide}</h2>
      <dl>
        <dt>{t.audience}</dt><dd>{guide.audience}</dd>
        <dt>{t.steps}</dt><dd>{guide.steps}</dd>
      </dl>
      <div className={styles.storyLinks}>
        <a href={guide.sourceUrl} target="_blank" rel="noopener noreferrer">
          {t.source} <ArrowUpRight size={14} aria-hidden="true" />
        </a>
      </div>
    </section>
  )
}
