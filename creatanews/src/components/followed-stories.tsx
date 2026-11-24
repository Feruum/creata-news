'use client'

import Link from 'next/link'
import { useEffect, useState } from 'react'
import type { FollowedStories as FollowedStoriesResponse } from '@creata/contracts'
import type { Locale } from '@/lib/models'
import { publicArticles } from '@/lib/content'
import { copy, localePath, localize } from '@/lib/i18n'
import { followedStorySummary, formatStoryTime } from '@/lib/story-content'
import { storyCopy } from '@/lib/story-copy'
import { acknowledgeStoryUpdates, waitForStoryReads } from '@/lib/story-reading'
import { useAuth } from './auth-context'
import { StateMessage } from './state-message'
import styles from './article-view.module.css'

type FollowRequest = { userId: string; data: FollowedStoriesResponse | null }
const noUpdates: readonly string[] = []

function useFollowedStories() {
  const { auth, loading, error, request, refresh } = useAuth()
  const userId = auth.user?.id
  const [result, setResult] = useState<FollowRequest | null>(null)
  const [attempt, setAttempt] = useState(0)

  useEffect(() => {
    if (!userId) return
    let active = true
    void waitForStoryReads(userId).then(() => request<FollowedStoriesResponse>('/account/followed-stories')).then(
      data => { if (active) setResult({ userId, data }) },
      () => { if (active) setResult({ userId, data: null }) },
    )
    return () => { active = false }
  }, [request, userId, attempt])

  function retry() {
    if (!userId) {
      void refresh()
      return
    }
    setResult(null)
    setAttempt(value => value + 1)
  }

  const data = result?.userId === userId ? result?.data : undefined
  return { userId, data, loading, error: error || data === null, request, retry }
}

export function FollowStoryButton({ id, slug, locale, updateIds }: {
  id: string
  slug: string
  locale: Locale
  updateIds: string[]
}) {
  const { userId, data, loading, error, request, retry } = useFollowedStories()
  const [change, setChange] = useState<{ userId: string; following: boolean }>()
  const [busy, setBusy] = useState(false)
  const [failed, setFailed] = useState(false)
  const t = storyCopy[locale]
  const following = change?.userId === userId ? change?.following : data?.items.some(article => article.id === id)
  const articlePath = localePath(locale, `/articles/${encodeURIComponent(slug)}`)
  const followPath = `/account/followed-stories/${encodeURIComponent(id)}`
  const unreadIds = data?.unreadUpdateIds?.[id] ?? noUpdates

  useEffect(() => {
    if (!userId || !following || !unreadIds.length || typeof IntersectionObserver === 'undefined') return
    const headings = new Map<Element, string>()
    for (const updateId of updateIds) {
      if (!unreadIds.includes(updateId)) continue
      const heading = document.getElementById(`story-update-${updateId}`)?.querySelector('h3')
      if (heading) headings.set(heading, updateId)
    }
    if (!headings.size) return

    const visible = new Set<Element>()
    const pending = new Set<Element>()
    function acknowledge() {
      if (document.visibilityState !== 'visible' || !visible.size) return
      const readHeadings = new Map<Element, string>()
      for (const heading of visible) {
        const updateId = headings.get(heading)
        if (!updateId || pending.has(heading)) continue
        readHeadings.set(heading, updateId)
        pending.add(heading)
      }
      if (!readHeadings.size || !userId) return
      const body = JSON.stringify({ updateIds: [...readHeadings.values()] })
      void acknowledgeStoryUpdates(userId, () => request(`${followPath}/read`, {
        method: 'POST', body, keepalive: true, signal: AbortSignal.timeout(3500),
      })).then(saved => {
        for (const heading of readHeadings.keys()) {
          pending.delete(heading)
          if (saved) {
            headings.delete(heading)
            visible.delete(heading)
            observer.unobserve(heading)
          }
        }
      })
    }
    const visibleFraction = 0.5
    const observer = new IntersectionObserver(entries => {
      for (const entry of entries) {
        if (entry.intersectionRatio >= visibleFraction) visible.add(entry.target)
        else visible.delete(entry.target)
      }
      acknowledge()
    }, { threshold: visibleFraction })
    for (const heading of headings.keys()) observer.observe(heading)
    document.addEventListener('visibilitychange', acknowledge)
    return () => {
      observer.disconnect()
      document.removeEventListener('visibilitychange', acknowledge)
    }
  }, [followPath, following, request, unreadIds, updateIds, userId])

  async function toggle() {
    if (!userId) return
    setBusy(true)
    setFailed(false)
    try {
      await request(followPath, { method: following ? 'DELETE' : 'POST' })
      setChange({ userId, following: !following })
    } catch {
      setFailed(true)
    } finally {
      setBusy(false)
    }
  }

  if (error) return <button className="button button-outline" onClick={retry}>{copy[locale].retry}</button>
  if (!loading && !userId) {
    return (
      <Link className="button button-outline" href={localePath(locale, `/profile?returnTo=${encodeURIComponent(articlePath)}`)}>
        {t.follow}
      </Link>
    )
  }

  return (
    <div className={styles.followControls}>
      <button className="button button-outline" disabled={loading || !data || busy} aria-pressed={!!following} onClick={toggle}>
        {following ? t.unfollow : t.follow}
      </button>
      {!!following && <Link href={localePath(locale, '/following')}>{t.mine}</Link>}
      {failed && <span className="inline-message error" role="alert">{copy[locale].operationFailed}</span>}
    </div>
  )
}

export function FollowedStories({ locale }: { locale: Locale }) {
  const { userId, data, loading, error } = useFollowedStories()
  const t = storyCopy[locale]
  if (error) return <StateMessage locale={locale} error />
  if (loading || (userId && !data)) return <div className="state-message" role="status">{copy[locale].loading}</div>
  if (!userId) {
    return (
      <div className="state-message">
        <p>{t.signIn}</p>
        <Link className="button button-outline" href={localePath(locale, `/profile?returnTo=${encodeURIComponent(localePath(locale, '/following'))}`)}>
          {copy[locale].signIn}
        </Link>
      </div>
    )
  }

  const stories = publicArticles(data?.items ?? [], locale).flatMap(article => {
    const summary = followedStorySummary(article.storyUpdates, data?.unreadUpdateIds?.[article.id] ?? noUpdates, locale)
    return summary ? [{ article, ...summary }] : []
  }).sort((left, right) => {
    const priority = Number(right.unreadCount > 0) - Number(left.unreadCount > 0)
    return priority || Date.parse(right.latest.occurredAt) - Date.parse(left.latest.occurredAt)
  })

  if (!stories.length) return <StateMessage locale={locale} title={t.empty} text={t.emptyHint} />
  return (
    <ol className={styles.followedList}>
      {stories.map(({ article, target, unreadCount }) => (
        <li key={article.id}>
          <div className={styles.updateMeta}>
            <time dateTime={target.occurredAt}>{formatStoryTime(target.occurredAt, locale)}</time>
            {unreadCount > 0 && <span className={styles.unreadLabel}>{t.unread} · {unreadCount}</span>}
          </div>
          <h2><Link href={localePath(locale, `/articles/${encodeURIComponent(article.slug)}#story-update-${target.id}`)}>{localize(article.title, locale)}</Link></h2>
          <p>{target.title}</p>
        </li>
      ))}
    </ol>
  )
}
