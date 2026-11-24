'use client'
import Link from 'next/link'
import { useId } from 'react'
import type { Article,Category,Locale } from '@/lib/models'
import { copy,localePath,localize } from '@/lib/i18n'
import { useCarousel } from '@/shared/hooks/use-carousel'
import { CarouselActions } from '@/shared/components/carousel-actions'
import { MediaImage } from '@/components/media-image'
import { CreataBrand } from '@creata/brand/logo'
import { CreataEmblem } from './creata-emblem'
import styles from './popular-stories.module.css'
import {useDemoArticles} from '@/demo/use-pipeline'

export function PopularStories({locale,articles:initialArticles,categories,imageForArticle,onNavigate}:{locale:Locale;articles:Article[];categories:Category[];imageForArticle?:(article:Article)=>string|undefined;onNavigate?:()=>void}) {
  const articles=useDemoArticles(initialArticles)
  const carousel=useCarousel<HTMLOListElement>(articles.length),trackId=useId(),title={ru:'Популярное',kk:'Танымал',en:'Most popular'}[locale],t=copy[locale]
  if(!articles.length)return null
  return <section className={styles.section} data-component='creata-popular' data-layout={articles.length===1?'single':articles.length===2?'pair':'carousel'} aria-labelledby={`${trackId}-title`}>
    <div className={styles.heading}>
      <div className={styles.signature} aria-hidden='true'><CreataBrand/></div>
      <h2 id={`${trackId}-title`}>{title}</h2>
      <div className={styles.brandArt}><CreataEmblem className={styles.emblem}/></div>
      <p className={styles.brandCaption}>{locale==='ru'?'То, что читают в Creata.':locale==='kk'?'Creata-да оқылатын оқиғалар.':'What’s being read at Creata.'}</p>
    </div>
    <div className={styles.body}>
      <div ref={carousel.viewport} id={trackId} className={styles.viewport} tabIndex={0} onKeyDown={carousel.keyboard} role='group' aria-roledescription='carousel' aria-label={title} data-current-index={carousel.selected} data-snap-count={carousel.snapCount}><ol ref={carousel.track} className={styles.track} role='list' aria-label={title}>
        {articles.map((article,index)=>{const category=categories.find(item=>item.id===article.categoryId);return <li key={article.id} className={styles.card}><Link href={localePath(locale,`/articles/${article.slug}`)} onNavigate={onNavigate}><MediaImage src={imageForArticle?.(article)??article.media?.url}/><span className={styles.shade}/><span className={styles.rank} aria-hidden='true'>{index+1}</span><div className={styles.copy}>{category&&<span className={styles.category}>{localize(category.name,locale)}</span>}<h3>{localize(article.title,locale)}</h3><p>{t.by} {article.author?.name||t.authorUnknown}</p></div></Link></li>})}
      </ol></div>
      <div className={styles.actions}><CarouselActions title={title} trackId={trackId} canScroll={carousel.canScroll} scroll={carousel.scroll} locale={locale}><span className={styles.period}>{locale==='ru'?'За неделю':locale==='kk'?'Осы аптада':'This week'}</span></CarouselActions></div>
    </div>
  </section>
}
