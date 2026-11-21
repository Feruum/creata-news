import type { ReactNode } from 'react'
import { copy,localize } from '@/lib/i18n'
import type { Article,Category,Locale } from '@/lib/models'
import { publicArticles } from '@/lib/content'
import { categoryTopic,topicDescription } from '@/lib/topics'
import { CollectionScreen } from '@/features/news/components/collection-screen'
export type ListingProps={locale:Locale;title:string;description?:string;introduction?:ReactNode;layer?:string;categoryId?:string;region?:string;q?:string;authorId?:string}
export function ArticleListingView({locale,title,description,introduction,layer,categoryId,region,articles,categories,error=false}:ListingProps&{articles:Article[];categories:Category[];error?:boolean}){const t=copy[locale],regions=[...new Set(articles.map(a=>a.region).filter(Boolean))];
  const category=categories.find(item=>item.id===categoryId),topic=category?categoryTopic(category):undefined
  const intro=description||(layer==='eco'?{ru:'Люди, команды и проекты экосистемы Creata. Истории о том, что мы создаём и как к этому присоединиться.',kk:'Creata экожүйесінің адамдары, командалары мен жобалары. Бірге не жасап жатқанымыз және қалай қосылуға болатыны туралы оқиғалар.',en:'People, teams and projects in the Creata ecosystem. What we are building and how to get involved.'}[locale]:topic?topicDescription(topic,locale):t.intro)
  const toolbar=<form className='listing-toolbar'>
    <label className='sr-only' htmlFor='category'>{t.topics}</label><select id='category' name='category' defaultValue={categoryId||''}><option value=''>{t.allTopics}</option>{categories.map(category=><option key={category.id} value={category.id}>{localize(category.name,locale)}</option>)}</select>
    <label className='sr-only' htmlFor='region'>{t.regions}</label><select id='region' name='region' defaultValue={region||''}><option value=''>{t.allRegions}</option>{regions.map(value=><option key={value} value={value}>{value}</option>)}</select><button className='button button-outline' type='submit'>{t.filters}</button><span className='listing-summary'>{articles.length} {t.articles.toLowerCase()}</span>
  </form>
  return <CollectionScreen locale={locale} title={title} description={intro} introduction={introduction} toolbar={toolbar} articles={articles} categories={categories} error={error}/>
}
