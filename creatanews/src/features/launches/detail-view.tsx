import Link from 'next/link'
import type {Locale,Launch,Category} from '@creata/contracts'
import {ArrowLeft,ArrowUpRight} from 'lucide-react'
import {copy,formatDate,localePath,localize} from '@/lib/i18n'
import {regionLabel} from '@/lib/topics'
import {MediaImage} from '@/components/media-image'
import {StoryCarousel} from '@/components/skiper'
import {launchCopy,launchText} from './copy'
import {LaunchCard} from './launch-card'
import {LaunchVoteButton,LaunchVotesProvider} from './votes'
import {LaunchDiscussion} from './discussion'
import styles from './launches.module.css'
export function LaunchDetail({locale,launch,others,categories}:{locale:Locale;launch:Launch;others:Launch[];categories:Category[]}){const l=launchCopy[locale],t=copy[locale];
  const related=others.filter(item=>item.id!==launch.id).slice(0,6),category=categories.find(item=>item.id===launch.categoryId)
  const tagline=launchText(launch.tagline,locale), product=launchText(launch.description,locale), team=launchText(launch.team,locale)
  return <LaunchVotesProvider locale={locale}><article>
    <Link className={styles.back} href={localePath(locale,'/launches')}><ArrowLeft size={16}/>{l.back}</Link>
    <header className={styles.header}><span className='eyebrow'>{l.title}</span><h1>{launch.name}</h1><p className={styles.tagline} lang={tagline.language}>{tagline.text}</p></header>
    <div className={styles.layout}>
      <div className={styles.content}>
        {launch.photo&&<MediaImage src={launch.photo} alt={`${l.photo}: ${launch.name}`} priority/>}
        <section><h2>{l.product}</h2><p className={styles.prose} lang={product.language}>{product.text}</p></section>
        <section><h2>{l.team}</h2><p className={styles.prose} lang={team.language}>{team.text}</p></section>
      </div>
      <aside className={styles.aside}>
        <dl className={styles.facts}><div><dt>{l.date}</dt><dd><time dateTime={launch.launchDate}>{formatDate(`${launch.launchDate}T12:00:00Z`,locale)}</time></dd></div>{launch.country&&<div><dt>{t.country}</dt><dd>{regionLabel(launch.country,locale)}</dd></div>}{category&&<div><dt>{t.topics}</dt><dd>{localize(category.name,locale)}</dd></div>}</dl>
        <a className='button button-accent' href={launch.url} target='_blank' rel='noopener noreferrer'>{l.website}<ArrowUpRight size={18}/></a>
        <div className={styles.detailVote}><LaunchVoteButton id={launch.id} slug={launch.slug} name={launch.name} initialVotes={launch.votes||0} locale={locale}/></div>
      </aside>
    </div>
    <LaunchDiscussion launchId={launch.id} slug={launch.slug} locale={locale}/>
    {related.length>0&&<section className={styles.related}><h2>{l.related}</h2><StoryCarousel locale={locale} label={l.related}>{related.map(item=><LaunchCard key={item.id} launch={item} locale={locale} categories={categories}/>)}</StoryCarousel></section>}
  </article></LaunchVotesProvider>
}
