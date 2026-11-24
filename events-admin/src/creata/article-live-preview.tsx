import { articleBodyBlocks, articlePhotoLayout, safeExternalUrl, type Article, type Locale } from '@creata/contracts'
import { formatDate, tr } from './i18n'
import { ArticlePhotoFigure } from './article-photos'

export function ArticleLivePreview({ article, language, label, publicWebsite }: {
  article: Article
  language: Locale
  label: string
  publicWebsite: string
}) {
  const text = (key: 'title' | 'lead' | 'summary' | 'body' | 'actions') => article[key]?.[language] || ''
  const summary = text('summary'), actions = text('actions')
  const blocks = articleBodyBlocks(text('body'))
  const { cover, inline } = articlePhotoLayout(article)
  const sources = article.sources?.flatMap(source => {
    const href = safeExternalUrl(source.url)
    return href ? [{ source, href }] : []
  }) ?? []
  const updates = (article.storyUpdates ?? []).flatMap(update => {
    const title = update.title[language]?.trim()
    const body = update.body[language]?.trim()
    const href = safeExternalUrl(update.sourceUrl)
    const timestamp = Date.parse(update.occurredAt)
    return title && body && href && Number.isFinite(timestamp) ? [{ ...update, title, body, href, timestamp }] : []
  }).sort((left, right) => right.timestamp - left.timestamp)
  const latest = updates[0]
  const audience = article.actionGuide?.audience[language]?.trim()
  const steps = article.actionGuide?.steps[language]?.trim()
  const guideSource = safeExternalUrl(article.actionGuide?.sourceUrl ?? '')
  const guide = audience && steps && guideSource ? { audience, steps, href: guideSource } : null
  const contents: { id: string; label: string }[] = []
  if (summary) contents.push({ id: 'summary', label: tr('articleBrief', language) })
  if (latest) contents.push({ id: 'timeline', label: tr('storyUpdates', language) })
  for (const block of blocks) {
    if (block.id) contents.push({ id: block.id, label: block.label || tr('body', language) })
  }
  if (actions) contents.push({ id: 'actions', label: tr('actions', language) })
  if (guide) contents.push({ id: 'action-guide', label: tr('storyGuide', language) })
  if (sources.length) contents.push({ id: 'sources', label: tr('facts', language) })

  return <section className='article-live-preview' id='article-live-preview' aria-label={label} lang={language}>
    <h3>{text('title') || tr('untitled', language)}</h3>
    {text('lead') && <p className='article-live-lead'>{text('lead')}</p>}
    {article.media?.url && inline.length > 0 && <figure className='article-photo'>
      <img src={article.media.url} alt={article.media.caption} referrerPolicy='no-referrer'/>
      <figcaption>{article.media.caption}{article.media.credit && ` · ${article.media.credit}`}</figcaption>
    </figure>}
    {cover && <ArticlePhotoFigure photo={cover} language={language} priority/>}
    {latest && <p className='article-live-latest'>
      <strong>{tr('storyLatest', language)}</strong>{' · '}
      <time dateTime={latest.occurredAt}>{formatDate(latest.occurredAt, language, true)}</time>
    </p>}
    {contents.length > 1 && <nav aria-label={tr('articleContents', language)}>
      <strong>{tr('articleContents', language)}</strong>
      <ul>{contents.map(item => <li key={item.id}>
        <a href={`#live-${item.id}`} onClick={event => {
          event.preventDefault()
          document.getElementById(`live-${item.id}`)?.scrollIntoView({ block: 'nearest' })
        }}>{item.label}</a>
      </li>)}</ul>
    </nav>}
    {summary && <p id='live-summary' className='article-live-summary'>{summary}</p>}
    {latest && <section id='live-timeline'>
      <h4>{tr('storyUpdates', language)}</h4>
      <ol className='article-live-updates'>{updates.map(update => <li key={update.id}>
        <time dateTime={update.occurredAt}>{formatDate(update.occurredAt, language, true)}</time>
        <h5>{update.title}</h5>
        <p>{update.body}</p>
        <div className='article-live-update-links'>
          <a className='external-link' href={update.href} target='_blank' rel='noopener noreferrer'>{tr('storyUpdateSource', language)}</a>
          {update.articleSlug?.trim() && <a className='external-link' href={`${publicWebsite}/${language}/articles/${encodeURIComponent(update.articleSlug.trim())}`}
            target='_blank' rel='noopener noreferrer'>{tr('storyRelatedArticle', language)}</a>}
        </div>
      </li>)}</ol>
    </section>}
    <div className='article-live-prose'>
      {blocks.length ? blocks.map(({ text, heading, id }, index) => (
        heading ? <h4 key={index} id={`live-${id}`}>{heading}</h4> : <p key={index} id={id ? `live-${id}` : undefined}>{text}</p>
      )) : <p className='muted'>{tr('livePreviewEmpty', language)}</p>}
    </div>
    {inline.map(photo => <ArticlePhotoFigure key={photo.url} photo={photo} language={language}/>)}
    {actions && <section id='live-actions'><h4>{tr('actions', language)}</h4><p>{actions}</p></section>}
    {guide && <section id='live-action-guide' className='article-live-guide'>
      <h4>{tr('storyGuide', language)}</h4>
      <dl>
        <dt>{tr('storyGuideAudience', language)}</dt><dd>{guide.audience}</dd>
        <dt>{tr('storyGuideSteps', language)}</dt><dd>{guide.steps}</dd>
      </dl>
      <a className='external-link' href={guide.href} target='_blank' rel='noopener noreferrer'>{tr('storyGuideSource', language)}</a>
    </section>}
    {sources.length > 0 && <section id='live-sources'>
      <h4>{tr('facts', language)}</h4>
      <ul className='article-live-sources'>{sources.map(({ source, href }, index) => <li key={`${href}-${index}`}>
        <strong>{source.title || new URL(href).hostname}</strong>
        {source.publishedAt && <time dateTime={source.publishedAt}>{formatDate(source.publishedAt, language)}</time>}
      </li>)}</ul>
    </section>}
  </section>
}
