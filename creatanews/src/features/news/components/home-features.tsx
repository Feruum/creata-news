'use client'

import Link from 'next/link'
import Image from 'next/image'
import {ArrowUpRight,Newspaper} from 'lucide-react'
import type {Article,Category,Locale} from '@/lib/models'
import {ArticleCard} from '@/components/cards'
import {localePath} from '@/lib/i18n'
import type {NewspaperIssue} from '@/features/newspaper/catalog'
import {newspaperCopy} from '@/features/newspaper/catalog'
import {demoMode} from '@/demo/config'
import {useDemoPipelineState} from '@/demo/use-pipeline'
import {demoNewspaperIssues} from '@/demo/newspapers'
import {homeLabels} from '../home-copy'

export function HomeFeatures({articles,categories,locale,issues}:{articles:Article[];categories:Category[];locale:Locale;issues:NewspaperIssue[]}){
 const l=homeLabels[locale],state=useDemoPipelineState()
 const editions=demoMode?state?demoNewspaperIssues(state):[]:issues,issue=editions[0]
 const photographed=articles.filter(article=>article.media?.url&&!/\.svg(?:\?|$)/i.test(article.media.url))
 const detailed=photographed.slice(1,3)
 return <>
  <section className='home-feature-band' aria-labelledby='home-feature-title'>
   <Image src='/images/home/creata-maker-v1.png' alt='' fill sizes='100vw'/>
   <div className='home-feature-copy'><h2 id='home-feature-title' data-home-reveal='fade'>{l.quote}</h2></div>
  </section>
  {detailed.length>0&&<section className='home-detailed-stories' aria-labelledby='home-month-heading'>
   <div className='section-heading'><h2 id='home-month-heading'>{l.month}</h2></div>
   <div className='home-pair-grid'>{detailed.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories} variant='editorial'/>)}</div>
  </section>}
  {issue&&<section className='home-newspaper-promo' aria-labelledby='home-newspaper-title' data-testid='home-newspaper-promo'>
   <div className='home-newspaper-copy' data-home-reveal='rise'>
    <div className='home-newspaper-kicker'><Newspaper size={22} aria-hidden='true'/><span>{l.newspaperKicker}</span></div>
    <h2 id='home-newspaper-title'>{l.newspaper}</h2><p>{l.newspaperText}</p>
    <Link href={localePath(locale,`/gazeta/${issue.id}`)}>{newspaperCopy[locale].read}<ArrowUpRight size={23} aria-hidden='true'/></Link>
   </div>
   <Link className='home-newspaper-cover' href={localePath(locale,`/gazeta/${issue.id}`)} tabIndex={-1} aria-hidden='true'><Image src={issue.cover} alt='' fill unoptimized sizes='(min-width: 1100px) 240px, (min-width: 900px) 185px, 160px'/></Link>
  </section>}
 </>
}
