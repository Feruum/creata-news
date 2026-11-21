import Link from 'next/link'
import { ArrowRight } from 'lucide-react'
import { publicArticles,buildQuery } from '@/lib/content'
import { copy,localePath,localize } from '@/lib/i18n'
import type { Article,Category,Locale } from '@/lib/models'
import { articleTopics,findTopic,topics,topicLabel,topicDescription,regionLabel } from '@/lib/topics'
import { ArticleCard } from './cards'
import { StateMessage } from './state-message'
import { CollectionScreen } from '@/features/news/components/collection-screen'

const labels={
  ru:{intro:'Новости, проекты и люди, за которыми стоит следить.',empty:'Материалы этой рубрики скоро появятся.',reset:'Сбросить фильтры'},
  kk:{intro:'Назар аударуға тұрарлық жаңалықтар, жобалар мен адамдар.',empty:'Бұл бөлімнің материалдары жақында жарияланады.',reset:'Сүзгілерді тазалау'},
  en:{intro:'News, projects and people to follow.',empty:'Stories in this section are on the way.',reset:'Clear filters'},
}

export function LatestView({locale,section,category,region,items,categories,error=false}:{locale:Locale;section?:string;category?:string;region?:string;items:Article[];categories:Category[];error?:boolean}) {
  const t=copy[locale],text=labels[locale]
  const articles=publicArticles(items,locale).sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||''))
  const selected=findTopic(section)
  const filtered=articles.filter(article=>(!region||article.region===region)&&(!category||article.categoryId===category)&&(!section||section==='recent'||!!selected&&articleTopics(article,categories).some(topic=>topic.id===selected.id)))
  if(section||category||region){
    const categoryName=localize(categories.find(item=>item.id===category)?.name,locale)
    const title=section==='recent'?undefined:selected?topicLabel(selected,locale):categoryName||(region?regionLabel(region,locale):t.noResults)
    return <CollectionScreen locale={locale} articles={filtered} categories={categories} title={title} description={selected?topicDescription(selected,locale):section==='recent'?undefined:text.intro} error={error}/>
  }
  const base=localePath(locale,'/latest')
  const href=(next?:string)=>base+buildQuery({section:next,region})
  const rows:{id:string;title:string;href:string;articles:Article[]}[]=[
    {id:'recent',title:t.latest,href:href('recent'),articles:filtered.slice(0,3)},
    ...topics.map(topic=>({id:topic.id,title:topicLabel(topic,locale),href:href(topic.id),articles:filtered.filter(article=>articleTopics(article,categories).some(item=>item.id===topic.id)).slice(0,3)})).filter(row=>row.articles.length>0),
  ]
  return <section className='listing-section latest-section'>
    <div className='page-heading'><p className='eyebrow'>Creata News</p><h1>{t.latest}</h1><p>{text.intro}</p></div>
    {error?<StateMessage locale={locale} error/>:filtered.length===0&&section!=='recent'&&(section||category||region)?<StateMessage locale={locale} title={t.noResults} text={t.searchHint}/>:articles.length===0?<StateMessage locale={locale}/>:<div className='topic-rows'>{rows.map(row=><section className='topic-row' key={row.id} aria-labelledby={`topic-${row.id}`}><h2 id={`topic-${row.id}`}><Link href={row.href}>{row.title}<ArrowRight size={23} aria-hidden='true'/></Link></h2><div className='topic-row-stories'>{row.articles.length?row.articles.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories}/>):<p className='topic-row-empty'>{text.empty}</p>}</div></section>)}</div>}
  </section>
}
