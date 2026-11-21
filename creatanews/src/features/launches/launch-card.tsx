import Link from 'next/link'
import { ArrowUpRight,MessageCircle,Tag } from 'lucide-react'
import type { Category, Launch, Locale } from '@creata/contracts'
import { MediaImage } from '@/components/media-image'
import { TextRoll } from '@/components/skiper'
import { formatDate, localePath, localize } from '@/lib/i18n'
import { regionLabel } from '@/lib/topics'
import { launchSocialCopy } from './copy'
import { LaunchVoteButton } from './votes'
import styles from './launches.module.css'

export function LaunchCard({launch,locale,categories=[],rank,variant='card'}:{launch:Launch;locale:Locale;categories?:Category[];rank?:number;variant?:'card'|'row'}) {
  const href=localePath(locale,`/launches/${launch.slug}`)
  const category=categories.find(item=>item.id===launch.categoryId)
  if(variant==='row')return <article className={styles.row}>
    <Link href={href} className={styles.logoLink} tabIndex={-1} aria-hidden='true'><MediaImage className={styles.logo} src={launch.logoUrl||launch.photo} fallback={<span className={styles.monogram}>{launch.name.charAt(0)}</span>}/></Link>
    <div className={styles.rowCopy}>
      <div className={styles.rowTitle}><h2>{rank&&<span>{rank}. </span>}<Link href={href}><TextRoll>{launch.name}</TextRoll></Link></h2><a href={launch.url} target='_blank' rel='noopener noreferrer' aria-label={`${launch.name} ↗`} className={styles.external}><ArrowUpRight size={18}/></a></div>
      <p>{localize(launch.tagline,locale)}</p>
      {!!(category||launch.tags?.length)&&<ul className={styles.tags} aria-label={launchSocialCopy[locale].tags}><li aria-hidden='true'><Tag size={16}/></li>{category&&<li><Link href={`${localePath(locale,'/launches')}?category=${encodeURIComponent(category.id)}`}>{localize(category.name,locale)}</Link></li>}{launch.tags?.map(tag=><li key={tag}><Link href={`${localePath(locale,'/launches')}?q=${encodeURIComponent(tag)}`}>{tag}</Link></li>)}</ul>}
    </div>
    <div className={styles.actions}><Link href={`${href}#discussion`} className={styles.counter} aria-label={`${launchSocialCopy[locale].discussion}: ${launch.name} (${launch.commentsCount||0})`}><MessageCircle size={17} aria-hidden='true'/><span>{launch.commentsCount||0}</span></Link><LaunchVoteButton id={launch.id} slug={launch.slug} name={launch.name} initialVotes={launch.votes||0} locale={locale}/></div>
  </article>
  return <article className={`article-card ${styles.card}`}>
    <Link href={href} className='card-image-link' tabIndex={-1} aria-hidden='true'><MediaImage src={launch.photo}/></Link>
    <div className='card-copy'>
      <div className='card-kicker'>{category&&localize(category.name,locale)}{launch.country&&<span>{regionLabel(launch.country,locale)}</span>}</div>
      <h2><Link href={href}><TextRoll>{launch.name}</TextRoll></Link></h2>
      <p>{localize(launch.tagline,locale)}</p>
      <div className='card-meta'><time dateTime={launch.launchDate}>{formatDate(`${launch.launchDate}T12:00:00Z`,locale)}</time></div>
    </div>
  </article>
}
