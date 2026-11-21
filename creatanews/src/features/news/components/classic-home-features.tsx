'use client'
import Link from 'next/link'
import Image from 'next/image'
import {ArrowUpRight} from 'lucide-react'
import type {Article,Category,Locale} from '@/lib/models'
import {ArticleCard} from '@/components/cards'
import {copy,localePath,localize} from '@/lib/i18n'
import type {NewspaperIssue} from '@/features/newspaper/catalog'
import {newspaperCopy} from '@/features/newspaper/catalog'
import {demoMode} from '@/demo/config'
import {useDemoPipelineState} from '@/demo/use-pipeline'
import {demoNewspaperIssues} from '@/demo/newspapers'
const labels={
 ru:{quote:'Будущее не случается. Его делают люди.',stories:'Истории в деталях',newspaper:'Главные идеи. В одном выпуске газеты.'},
 kk:{quote:'Болашақ өздігінен келмейді. Оны адамдар жасайды.',stories:'Оқиғаларға тереңірек үңіліңіз',newspaper:'Басты идеялар. Газеттің бір санында.'},
 en:{quote:'The future doesn’t just happen. People make it.',stories:'A closer look',newspaper:'The big ideas. In one newspaper.'},
}
export function ClassicHomeFeatures({articles,categories,locale,issues}:{articles:Article[];categories:Category[];locale:Locale;issues:NewspaperIssue[]}){
 const t=copy[locale],l=labels[locale],state=useDemoPipelineState(),editions=demoMode?state?demoNewspaperIssues(state):[]:issues,issue=editions[0]
 const photographed=articles.filter(article=>article.media?.url&&!/\.svg(?:\?|$)/i.test(article.media.url)),feature=photographed[0],detailed=photographed.slice(1,3)
 return <>
  {feature&&<section className='classic-home-feature-band' aria-labelledby='classic-home-feature-title'><Image src={feature.media!.url} alt='' fill unoptimized sizes='(min-width: 1440px) 1320px, 100vw'/><div className='classic-home-feature-copy'><h2 id='classic-home-feature-title'>{l.quote}</h2><Link href={localePath(locale,`/articles/${feature.slug}`)}>{t.readStory}<ArrowUpRight size={16}/></Link></div><p className='classic-home-feature-caption'>{localize(feature.title,locale)}</p></section>}
  {detailed.length>0&&<section className='original-news-section classic-home-detailed-stories'><div className='section-heading'><h2>{l.stories}</h2><Link href={localePath(locale,'/latest')}>{t.allStories}<ArrowUpRight size={16}/></Link></div><div className='story-grid classic-home-pair-grid'>{detailed.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories}/>)}</div></section>}
  {issue&&<section className='classic-home-newspaper-promo' aria-labelledby='classic-home-newspaper-title' data-testid='classic-home-newspaper-promo'><div className='classic-home-newspaper-copy'><h2 id='classic-home-newspaper-title'>{l.newspaper}</h2><p>{issue.headline}</p><Link className='button button-accent' href={localePath(locale,`/gazeta/${issue.id}`)}>{newspaperCopy[locale].read}<ArrowUpRight size={16}/></Link></div><Link className='classic-home-newspaper-cover' href={localePath(locale,`/gazeta/${issue.id}`)} tabIndex={-1} aria-hidden='true'><Image src={issue.cover} alt='' fill unoptimized sizes='(min-width: 900px) 280px, 180px'/></Link></section>}
 </>
}
