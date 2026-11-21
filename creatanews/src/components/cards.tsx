import Link from 'next/link'
import { ArrowUpRight } from 'lucide-react'
import type { Article, Category, Locale } from '@/lib/models'
import { copy, formatDate, localePath, localize } from '@/lib/i18n'
import { readingMinutes } from '@/lib/content'
import { MediaImage } from './media-image'
export function ArticleCard({article,locale,categories=[],variant='standard',priority=false,imageSrc}: {article:Article;locale:Locale;categories?:Category[];variant?:'standard'|'lead'|'compact'|'horizontal'|'editorial';priority?:boolean;imageSrc?:string}) {
  const t=copy[locale],category=categories.find(item=>item.id===article.categoryId), href=localePath(locale,`/articles/${article.slug}`),Heading=variant==='lead'?'h1':'h2'
  return <article className={`article-card article-card-${variant}`} data-home-reveal={variant==='editorial'?'card':undefined}>
    {variant!=='compact'&&<Link href={href} className='card-image-link' tabIndex={-1} aria-hidden='true'><MediaImage src={imageSrc??article.media?.url} priority={priority}/></Link>}
    <div className='card-copy'><div className='card-kicker'><span className='card-category'>{article.layer==='eco'?t.ecosystem:category?localize(category.name,locale):t.world}</span>{variant==='editorial'?<span>{readingMinutes(localize(article.body,locale))} {t.minRead}</span>:article.region&&<span>{article.region}</span>}</div><Heading><Link href={href}>{localize(article.title,locale)}</Link></Heading>{(variant==='lead'||variant==='editorial')&&<p className='card-lead'>{localize(article.lead,locale)}</p>}<div className='card-meta'>{variant==='editorial'&&<span className='card-author-mark' aria-hidden='true'>≡</span>}{variant!=='compact'&&<span className='card-author-name'>{variant!=='editorial'&&`${t.by} `}{article.author?.name||t.authorUnknown}</span>}<time dateTime={article.publishedAt}>{formatDate(article.publishedAt,locale,true)}</time>{variant!=='editorial'&&<span>{readingMinutes(localize(article.body,locale))} {t.minRead}</span>}</div></div>
  </article>
}
export { OpportunityCard } from '@/features/opportunities/opportunity-card'
export function Newsletter({locale}: {locale:Locale}) {const t=copy[locale];return <section className='newsletter-band'><div className='newsletter-art' aria-hidden='true'><span className='art-circle'/><span className='art-arrow'>↗</span><span className='art-word'>NEXT<br/>IS NOW.</span></div><div className='newsletter-copy'><span className='eyebrow'>{t.digest}</span><h2>{t.newsletterTitle}</h2><p>{t.newsletterText}</p><Link href={localePath(locale,'/profile#digest')} className='button button-accent'>{t.subscribe}<ArrowUpRight size={18}/></Link><small>{t.newsletterFine}</small></div></section>}
