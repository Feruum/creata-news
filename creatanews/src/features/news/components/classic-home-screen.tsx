'use client'

// Existing data adapters, links, carousel interactions and opportunities remain
// owned here; the presentation follows the requested editorial reference.
import { ArrowRight,ChevronRight } from 'lucide-react'
import Image from 'next/image'
import Link from 'next/link'
import { useId,useState } from 'react'
import {useDemoArticles} from '@/demo/use-pipeline'
import type { Article,Category,Locale,Opportunity } from '@/lib/models'
import { copy,formatDate,localePath } from '@/lib/i18n'
import { CarouselActions } from '@/shared/components/carousel-actions'
import { useCarousel } from '@/shared/hooks/use-carousel'
import { StateMessage } from '@/components/state-message'
import { ArticleCard } from '@/components/cards'
import {CreataBrand} from '@creata/brand/logo'
import {homeStoryImage} from '../home-story-images'
import { OpportunitiesBand } from '@/features/opportunities/opportunities-band'
import { adaptArticleForHome,type HomeStory } from '../home-adapter'
import {articleTopics,topics,topicLabel} from '@/lib/topics'
import type {NewspaperIssue} from '@/features/newspaper/catalog'
import {ClassicHomeFeatures} from './classic-home-features'
import styles from './classic-home-screen.module.css'

const homeCopy={ru:{spotlight:'В центре внимания',lead:'Главная история',more:'Ещё истории',latest:'Последние новости'},kk:{spotlight:'Назарда',lead:'Басты оқиға',more:'Тағы оқиғалар',latest:'Соңғы жаңалықтар'},en:{spotlight:'In the spotlight',lead:'The lead story',more:'More stories',latest:'Latest news'}}

function StoryArtwork({story,hero=false,eager=false}:{story:HomeStory;hero?:boolean;eager?:boolean}){
  const[failedImage,setFailedImage]=useState<string>()
  const image=story.imageUrl
  return image&&image!==failedImage?<Image src={image} alt={story.imageAlt} fill unoptimized sizes={hero?'(min-width: 1100px) 650px, (min-width: 700px) 65vw, 100vw':'(min-width: 900px) 340px, 85vw'} loading={hero||eager?'eager':'lazy'} onError={()=>setFailedImage(image)}/>:<div className={styles.artworkFallback} aria-hidden='true'><CreataBrand/></div>
}
function Byline({story,locale}:{story:HomeStory;locale:Locale}){const t=copy[locale];return <div className={styles.byline}><span>{t.by} {story.authorId?<Link href={localePath(locale,`/authors/${story.authorId}`)}>{story.authorName||t.authorUnknown}</Link>:story.authorName||t.authorUnknown}</span><span>{story.readMinutes} {t.minRead}</span></div>}
function Spotlight({story,locale}:{story?:HomeStory;locale:Locale}){
  const t=copy[locale]
  return <article className={`home-spotlight ${styles.lead}`} aria-labelledby='home-spotlight-title' data-testid='home-lead'>
    {story&&<><Link href={localePath(locale,`/articles/${story.slug}`)} className={styles.leadImage} tabIndex={-1}><StoryArtwork story={story} hero/></Link>{story.imageCredit&&<p className={styles.credit}>{story.imageCredit}</p>}<Link className={styles.kicker} href={`${localePath(locale,'/latest')}?category=${encodeURIComponent(story.categoryId)}`}>{story.layer==='eco'?t.ecosystem:story.categoryName||t.world}</Link></>}
    <h1 id='home-spotlight-title'>{story?<Link href={localePath(locale,`/articles/${story.slug}`)}>{story.title}</Link>:t.tagline}</h1>
    <p className={styles.deck}>{story?.lead||t.intro}</p>
    {story&&<Byline story={story} locale={locale}/>}
  </article>
}
function LatestRail({stories,locale}:{stories:HomeStory[];locale:Locale}){const t=copy[locale],text=homeCopy[locale];return <aside className={styles.latestRail} aria-labelledby='home-latest-heading' data-testid='home-latest'><h2 id='home-latest-heading'>{text.latest}</h2><ol>{stories.slice(0,8).map(story=><li key={story.id}><Link href={localePath(locale,`/articles/${story.slug}`)}>{story.title}</Link><p><time dateTime={story.publishedAt}>{formatDate(story.publishedAt,locale)}</time><span>{story.readMinutes} {t.minRead}</span></p></li>)}</ol><Link className={styles.allLatest} href={localePath(locale,'/latest')}>{t.allStories}<ArrowRight size={15}/></Link></aside>}
function SecondaryStories({stories,locale}:{stories:HomeStory[];locale:Locale}){const t=copy[locale];return <aside className={styles.secondaryStories} aria-label={homeCopy[locale].more} data-testid='home-secondary'>{stories.map(story=><article key={story.id}><Link href={localePath(locale,`/articles/${story.slug}`)} className={styles.secondaryImage} tabIndex={-1}><StoryArtwork story={story}/></Link><span className={styles.kicker}>{story.categoryName||t.world}</span><h2><Link href={localePath(locale,`/articles/${story.slug}`)}>{story.title}</Link></h2><Byline story={story} locale={locale}/></article>)}</aside>}
function PosterCard({story,featured=false,locale}:{story:HomeStory;featured?:boolean;locale:Locale}){
  const t=copy[locale]
  return <article className={styles.poster}><Link href={localePath(locale,`/articles/${story.slug}`)} className={styles.posterLink}>
    <div className={styles.posterArtwork}><StoryArtwork story={story} eager={featured}/></div><span className={styles.posterCategory}>{story.layer==='eco'?t.ecosystem:story.categoryName||t.world}</span>
    <div className={styles.posterCopy}><time dateTime={story.publishedAt}>{formatDate(story.publishedAt,locale,true)} · {story.readMinutes} {t.minRead}</time><h3>{story.title}</h3>{story.lead&&<p>{story.lead}</p>}<div className={styles.hosts}>{story.authorName||t.authorUnknown}</div></div><span className={styles.posterArrow} aria-hidden='true'><ArrowRight size={17}/></span>
  </Link></article>
}
export function ClassicHomeScreen({articles:initialArticles,categories,opportunities,locale,issues=[],error=false,opportunitiesError=false}:{articles:Article[];categories:Category[];opportunities:Opportunity[];locale:Locale;issues?:NewspaperIssue[];error?:boolean;opportunitiesError?:boolean}){
  const articles=useDemoArticles(initialArticles)
  const t=copy[locale],text=homeCopy[locale],stories=error?[]:articles.map(article=>adaptArticleForHome(article,locale,categories)),carousel=useCarousel(stories.length),railId=useId()
  const eco=articles.filter(article=>article.layer==='eco'),world=articles.filter(article=>article.layer==='world')
  const sections=topics.map(topic=>({topic,items:world.filter(article=>articleTopics(article,categories).some(item=>item.id===topic.id)).slice(0,3)})).filter(section=>section.items.length)
  return <div className={styles.home} data-testid='classic-home-screen'>
    <div className={styles.frontpage} data-testid='home-frontpage'><Spotlight story={stories[0]} locale={locale}/>{stories.length>0&&<><LatestRail stories={stories} locale={locale}/><SecondaryStories stories={stories.slice(1,3)} locale={locale}/></>}</div>
    {error?<StateMessage locale={locale} error/>:stories.length===0?<StateMessage locale={locale}/>:<section aria-label={text.spotlight}>
      <div className={`home-spotlight-actions ${styles.spotlightActions}`}><h2>{text.spotlight}</h2><CarouselActions title={text.spotlight} trackId={railId} canScroll={carousel.canScroll} scroll={carousel.scroll} locale={locale}><Link href={localePath(locale,'/latest')}>{t.allStories}<ChevronRight size={14}/></Link></CarouselActions></div>
      <div id={railId} ref={carousel.viewport} className={styles.spotlightViewport} tabIndex={0} onKeyDown={carousel.keyboard} aria-label={text.spotlight} role='group' aria-roledescription='carousel' data-current-index={carousel.selected} data-snap-count={carousel.snapCount}><div data-testid='home-story-rail' ref={carousel.track} className={styles.spotlightRail}>{stories.map((story,index)=><PosterCard key={story.id} story={story} featured={index===0} locale={locale}/>)}</div></div>
    </section>}
    {sections.map(({topic,items})=><section className='original-news-section' key={topic.id}><div className='section-heading'><h2><Link href={`${localePath(locale,'/latest')}?section=${topic.id}`}>{topicLabel(topic,locale)} <ArrowRight size={23}/></Link></h2><Link href={`${localePath(locale,'/latest')}?section=${topic.id}`}>{t.seeAll}<ChevronRight size={14}/></Link></div><div className='story-grid'>{items.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories} imageSrc={homeStoryImage(article)}/>)}</div></section>)}
    {eco.length>0&&<section className='original-news-section'><div className='section-heading'><h2>{t.ecosystem}</h2><Link href={localePath(locale,'/ecosystem')}>{t.seeAll}<ChevronRight size={14}/></Link></div><div className='story-grid'>{eco.slice(0,3).map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories} imageSrc={homeStoryImage(article)}/>)}</div></section>}
    <OpportunitiesBand opportunities={opportunities} locale={locale} error={opportunitiesError}/>
    <ClassicHomeFeatures articles={articles} categories={categories} locale={locale} issues={issues}/>
  </div>
}
