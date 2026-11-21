import Link from 'next/link'
import {Search,ArrowUpRight} from 'lucide-react'
import type {Article,Author,Category,Locale,Opportunity} from '@/lib/models'
import {copy,localePath,localize} from '@/lib/i18n'
import {matchSearchGroups,searchAvailability} from '@/lib/search-results'
import {ArticleCard,OpportunityCard} from '@/components/cards'
import {StateMessage} from '@/components/state-message'
export function SearchView({locale,q,articles,authors,categories,opportunities,errors=[false]}:{locale:Locale;q:string;articles:Article[];authors:Author[];categories:Category[];opportunities:Opportunity[];errors?:boolean[]}){const t=copy[locale],matched=matchSearchGroups({authors,categories,opportunities},q,locale),resultCount=articles.length+matched.authors.length+matched.categories.length+matched.opportunities.length,availability=searchAvailability(errors,resultCount);
  return <section className='listing-section'>
    <div className='page-heading'><span className='eyebrow'>Creata News</span><h1>{t.search}</h1>
      <form className='search-form'><label className='sr-only' htmlFor='search-query'>{t.search}</label><input name='q' id='search-query' defaultValue={q} maxLength={200} placeholder={t.searchPlaceholder}/><button className='button button-accent'><span>{t.search}</span><Search size={18}/></button></form>
    </div>
    {!q?<StateMessage locale={locale} title={t.explore} text={t.searchHint}/>:availability.failed?<StateMessage locale={locale} error/>:<>
      <h2 className='sr-only'>{t.results}</h2>
      {availability.partial&&<StateMessage locale={locale} error compact title={t.partialResults} text={t.partialResultsText}/>}
      {availability.empty&&<StateMessage locale={locale} title={t.noResults} text={t.searchHint}/>}
      {articles.length>0&&<section className='search-results-section'><h2>{t.articles} · {articles.length}</h2><div className='story-grid'>{articles.map(article=><ArticleCard key={article.id} article={article} locale={locale} categories={categories}/>)}</div></section>}
      {matched.categories.length>0&&<section className='search-results-section category-results'><h2>{t.topics}</h2><div className='category-result-grid'>{matched.categories.map(category=><Link key={category.id} href={localePath(locale,`/categories/${category.slug||category.id}`)}><span>{localize(category.name,locale)}</span><ArrowUpRight size={18}/></Link>)}</div></section>}
      {matched.authors.length>0&&<section className='search-results-section'><h2>{t.authors}</h2><div className='author-grid'>{matched.authors.map(author=><Link className='author-card' key={author.id} href={localePath(locale,`/authors/${author.id}`)}><div className='author-avatar'>{author.name.charAt(0)}</div><h3>{author.name}</h3></Link>)}</div></section>}
      {matched.opportunities.length>0&&<section className='search-results-section'><h2>{t.opportunities}</h2><div className='prototype-opportunity-grid'>{matched.opportunities.map(item=><OpportunityCard key={item.id} opportunity={item} locale={locale}/>)}</div></section>}
    </>}
  </section>
}
