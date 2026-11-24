import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import { articlePhotoLayout } from '@creata/contracts'
import type { Article,Category,Locale,Opportunity } from '@/lib/models'
import { copy,formatDate,localePath,localize } from '@/lib/i18n'
import { articleBodyBlocks,readingMinutes,safeExternalUrl } from '@/lib/content'
import { MediaImage } from './media-image'
import { ReadingProgress,StoryCarousel } from './skiper'
import { ArticleCard } from './cards'
import { SaveButton,ShareButton,ViewEvent } from './article-actions'
import { ArticleContents } from './article-contents'
import { ArticlePhotoFigure } from './article-photo'
import { ArticleActionGuide, StoryTimeline } from './article-story'
import { localizedActionGuide, localizedStoryUpdates } from '@/lib/story-content'
import { storyCopy } from '@/lib/story-copy'
import {OpportunityCard} from '@/features/opportunities/opportunity-card'
import {GraphicFigure} from '@creata/graphics/figure'
import {validateDataset} from '@creata/graphics'
import '@creata/graphics/styles.css'
import styles from './article-view.module.css'

export function ArticleView({article,locale,categories=[],related=[],opportunities=[],preview=false}: {article:Article;locale:Locale;categories?:Category[];related?:Article[];opportunities?:Opportunity[];preview?:boolean}) {
  const t = copy[locale], category = categories.find(item => item.id === article.categoryId)
  const body = localize(article.body, locale), summary = localize(article.summary, locale), actions = localize(article.actions, locale)
  const updates = localizedStoryUpdates(article.storyUpdates, locale)
  const guide = localizedActionGuide(article.actionGuide, locale)
  const story = storyCopy[locale]
  const blocks = articleBodyBlocks(body)
  const photos = articlePhotoLayout(article)
  const sources = article.sources.flatMap(source => {
    const href = safeExternalUrl(source.url)
    return href ? [{ source, href }] : []
  })
  const contents: { id: string; label: string }[] = []
  if (summary) contents.push({ id: 'article-summary', label: t.articleBrief })
  if (updates.length) contents.push({ id: 'article-timeline', label: story.timeline })
  for (const block of blocks) {
    if (block.id) contents.push({ id: block.id, label: block.label || t.articleText })
  }
  if (guide) contents.push({ id: 'article-guide', label: story.guide })
  if (actions) contents.push({ id: 'article-action', label: t.action })
  if (opportunities.length) contents.push({ id: 'article-opportunities', label: t.opportunities })
  if (sources.length) contents.push({ id: 'article-sources', label: t.sources })
  if (!preview && article.corrections?.length) contents.push({ id: 'article-corrections', label: t.corrections })

  return <>
  {preview?<div className='preview-banner'><strong>{t.preview}</strong>{t.previewNote}</div>:<ViewEvent id={article.id}/>}
  <header className='article-header'><div className='card-kicker'><Link href={localePath(locale,article.layer==='eco'?'/ecosystem':'/latest')}>{article.layer==='eco'?t.ecosystem:t.latest}</Link>{category&&<Link href={localePath(locale,`/categories/${category.slug||category.id}`)}>{localize(category.name,locale)}</Link>}{article.region&&<span>{article.region}</span>}</div><h1>{localize(article.title,locale)||t.unavailableTranslation}</h1><p className='article-lead'>{localize(article.lead,locale)}</p><div className='article-byline'><span>{t.by} <Link href={localePath(locale,`/authors/${article.authorId}`)}>{article.author?.name||t.authorUnknown}</Link></span><time dateTime={article.publishedAt}>{formatDate(article.publishedAt||article.createdAt,locale)}</time><span>{readingMinutes(body)} {t.minRead}</span></div></header>
  {article.media?.url&&<figure className='article-cover'><MediaImage src={article.media.url} alt={article.media.caption} priority/><figcaption>{article.media.caption}{article.media.credit&&` · ${article.media.credit}`}</figcaption></figure>}
  {photos.cover&&<ArticlePhotoFigure photo={photos.cover} locale={locale} cover/>}
  {article.format==='graphic'&&article.graphic&&!validateDataset(article.graphic)&&<div style={{maxWidth:1000,margin:'24px auto',padding:'0 20px'}}><GraphicFigure data={article.graphic} locale={locale} title={localize(article.title,locale)}/></div>}
  <div className='article-body-layout'>
    <aside className='article-sidebar'>
      {contents.length > 1 && <ArticleContents key={`${article.id}:${locale}`} label={t.articleContents} sections={contents}/>}
      {!preview&&<SaveButton id={article.id} type='article' locale={locale}/>}
      <ShareButton locale={locale}/>
    </aside><div>
    {summary&&<div id='article-summary' className='article-summary'>{summary}</div>}
    {updates.length > 0&&<StoryTimeline updates={updates} articleId={article.id} slug={article.slug} locale={locale} preview={preview}/>}
    <div className='article-prose'>{blocks.length ? blocks.map(({ text, heading, id }, index) => (
      heading ? <h2 key={index} id={id} className={styles.proseHeading}>{heading}</h2> : <p key={index} id={id}>{text}</p>
    )) : <p>{t.unavailableTranslation}</p>}</div>
    {photos.inline.map(photo => <ArticlePhotoFigure key={photo.url} photo={photo} locale={locale}/>)}
    {guide&&<ArticleActionGuide guide={guide} locale={locale}/>}
    {actions&&<section id='article-action' className='article-action'><h2>{t.action} ↗</h2><p>{actions}</p></section>}
    {opportunities.length>0&&<section id='article-opportunities' className='editorial-section'><div className='section-heading'><h2>{t.opportunities}<span>↗</span></h2></div><div className='prototype-opportunity-grid'>{opportunities.map((item,index)=><OpportunityCard key={item.id} opportunity={item} locale={locale} index={index}/>)}</div></section>}
    <section id='article-sources' className='article-sources'>
      <h2>{t.sources}</h2>
      {sources.map(({ source, href }, index) => {
        const date = formatDate(source.publishedAt, locale)
        return <a href={href} key={`${href}-${index}`} target='_blank' rel='noopener noreferrer'>
          <div className={styles.sourceDetails}>
            <strong>{source.title || new URL(href).hostname}</strong>
            {date && <time dateTime={source.publishedAt}>{date}</time>}
          </div>
          <ArrowUpRight className={styles.sourceIcon} size={17} aria-hidden='true'/>
        </a>
      })}
    </section>
    {!preview&&<section id='article-corrections' className='corrections'><h2>{t.corrections}</h2>{article.corrections?.length?<ul>{article.corrections.map((correction,index)=><li key={index}><time dateTime={correction.publishedAt}>{formatDate(correction.publishedAt,locale)}</time> — {correction.note}</li>)}</ul>:<p>{t.noCorrections}</p>}</section>}
  </div></div>
  {related.length>0&&<section className='editorial-section'><div className='section-heading'><h2>{t.related}<span>↗</span></h2></div><StoryCarousel locale={locale}>{related.map(item=><ArticleCard key={item.id} article={item} locale={locale} categories={categories}/>)}</StoryCarousel></section>}
  <ReadingProgress locale={locale}/>
 </>}
