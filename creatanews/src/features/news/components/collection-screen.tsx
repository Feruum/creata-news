// The original ClubProfileScreen's cover → identity → content structure,
// adapted to published news. The cover and back-link styles retain their owners'
// sizing, object-fit and responsive behavior; no club data or profile store is used.
import Link from 'next/link'
import type { ReactNode } from 'react'
import { ArrowLeft,ArrowUpRight } from 'lucide-react'
import type { Article,Category,Locale } from '@/lib/models'
import { copy,localePath } from '@/lib/i18n'
import { ArticleCard } from '@/components/cards'
import { MediaImage } from '@/components/media-image'
import { StateMessage } from '@/components/state-message'
import styles from './collection-screen.module.css'

const labels={
  ru:{collection:'Подборка',intro:'Свежие истории о технологиях, идеях и людях, которые меняют мир. Все последние материалы Creata News в одной подборке.'},
  kk:{collection:'Жинақ',intro:'Әлемді өзгертетін технологиялар, идеялар мен адамдар туралы жаңа оқиғалар. Creata News-тің соңғы материалдары бір жинақта.'},
  en:{collection:'Collection',intro:'Fresh stories about technology, ideas and people shaping the world. The latest from Creata News, together in one collection.'},
}

export function CollectionScreen({locale,articles,categories,title,description,introduction,toolbar,error=false}:{locale:Locale;articles:Article[];categories:Category[];title?:string;description?:string;introduction?:ReactNode;toolbar?:ReactNode;error?:boolean}) {
  const t=copy[locale],text=labels[locale],cover=articles.find(article=>article.media?.url)?.media
  return <article className={styles.page}>
    <header className={styles.header}>
      <div className={styles.cover}><MediaImage src={cover?.url} priority/><div className={styles.shade}/></div>
      <div className={styles.identity}><Link href={localePath(locale,'/latest')} className={styles.back}><ArrowLeft size={16} aria-hidden='true'/>{t.latest}</Link><span className={styles.badge}>{text.collection}</span><h1>{title||t.latest}</h1></div>
    </header>
    <div className={styles.content}>
      <div className={styles.summary}><p>{description||text.intro}</p>{introduction}<Link href={localePath(locale,'/profile#digest')}>{t.subscribe}<ArrowUpRight size={17} aria-hidden='true'/></Link></div>
      {toolbar}
      {error?<StateMessage locale={locale} error/>:articles.length?<section className={styles.stories} aria-label={title||t.latest}>{articles.map(article=><ArticleCard article={article} locale={locale} categories={categories} variant='horizontal' key={article.id}/>)}</section>:<StateMessage locale={locale}/>}
    </div>
  </article>
}
