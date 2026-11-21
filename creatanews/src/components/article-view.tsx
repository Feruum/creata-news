import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import type { Article,Category,Locale,Opportunity } from '@/lib/models'
import { copy,formatDate,localePath,localize } from '@/lib/i18n'
import { readingMinutes,safeExternalUrl } from '@/lib/content'
import { MediaImage } from './media-image'
import { ReadingProgress,StoryCarousel } from './skiper'
import { ArticleCard } from './cards'
import { SaveButton,ShareButton,ViewEvent } from './article-actions'
import {OpportunityCard} from '@/features/opportunities/opportunity-card'
import {GraphicFigure} from '@creata/graphics/figure'
import {validateDataset} from '@creata/graphics'
import '@creata/graphics/styles.css'
export function ArticleView({article,locale,categories=[],related=[],opportunities=[],preview=false}: {article:Article;locale:Locale;categories?:Category[];related?:Article[];opportunities?:Opportunity[];preview?:boolean}) {const t=copy[locale],category=categories.find(item=>item.id===article.categoryId),body=localize(article.body,locale);return <>
  {preview?<div className='preview-banner'><strong>{t.preview}</strong>{t.previewNote}</div>:<ViewEvent id={article.id}/>}
  <header className='article-header'><div className='card-kicker'><Link href={localePath(locale,article.layer==='eco'?'/ecosystem':'/latest')}>{article.layer==='eco'?t.ecosystem:t.latest}</Link>{category&&<Link href={localePath(locale,`/categories/${category.slug||category.id}`)}>{localize(category.name,locale)}</Link>}{article.region&&<span>{article.region}</span>}</div><h1>{localize(article.title,locale)||t.unavailableTranslation}</h1><p className='article-lead'>{localize(article.lead,locale)}</p><div className='article-byline'><span>{t.by} <Link href={localePath(locale,`/authors/${article.authorId}`)}>{article.author?.name||t.authorUnknown}</Link></span><time dateTime={article.publishedAt}>{formatDate(article.publishedAt||article.createdAt,locale)}</time><span>{readingMinutes(body)} {t.minRead}</span></div></header>
  {article.media?.url&&<figure className='article-cover'><MediaImage src={article.media.url} alt={article.media.caption} priority/><figcaption>{article.media.caption}{article.media.credit&&` · ${article.media.credit}`}</figcaption></figure>}
  {article.format==='graphic'&&article.graphic&&!validateDataset(article.graphic)&&<div style={{maxWidth:1000,margin:'24px auto',padding:'0 20px'}}><GraphicFigure data={article.graphic} locale={locale} title={localize(article.title,locale)}/></div>}
  <div className='article-body-layout'><aside className='article-sidebar'>{!preview&&<SaveButton id={article.id} type='article' locale={locale}/>}<ShareButton locale={locale}/></aside><div>
    {localize(article.summary,locale)&&<div className='article-summary'>{localize(article.summary,locale)}</div>}
    <div className='article-prose'>{body?body.split(/\n\s*\n|\n/).filter(Boolean).map((paragraph,index)=><p key={index}>{paragraph}</p>):<p>{t.unavailableTranslation}</p>}</div>
    {localize(article.actions,locale)&&<section className='article-action'><h2>{t.action} ↗</h2><p>{localize(article.actions,locale)}</p></section>}
    {opportunities.length>0&&<section className='editorial-section'><div className='section-heading'><h2>{t.opportunities}<span>↗</span></h2></div><div className='prototype-opportunity-grid'>{opportunities.map((item,index)=><OpportunityCard key={item.id} opportunity={item} locale={locale} index={index}/>)}</div></section>}
    <section className='article-sources'><h2>{t.sources}</h2>{article.sources.filter(source=>safeExternalUrl(source.url)).map((source,index)=><a href={safeExternalUrl(source.url)!} key={`${source.url}-${index}`} target='_blank' rel='noopener noreferrer'><div>{source.title||new URL(source.url).hostname}<span>{source.publishedAt&&formatDate(source.publishedAt,locale)}</span></div><ArrowUpRight size={17}/></a>)}</section>
    {!preview&&<section className='corrections'><h2>{t.corrections}</h2>{article.corrections?.length?<ul>{article.corrections.map((correction,index)=><li key={index}><time dateTime={correction.publishedAt}>{formatDate(correction.publishedAt,locale)}</time> — {correction.note}</li>)}</ul>:<p>{t.noCorrections}</p>}</section>}
  </div></div>
  {related.length>0&&<section className='editorial-section'><div className='section-heading'><h2>{t.related}<span>↗</span></h2></div><StoryCarousel locale={locale}>{related.map(item=><ArticleCard key={item.id} article={item} locale={locale} categories={categories}/>)}</StoryCarousel></section>}
  <ReadingProgress locale={locale}/>
 </>}
