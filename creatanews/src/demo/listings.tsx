'use client'
import type {Locale} from '@/lib/models'
import {ArticleListingView,type ListingProps} from '@/components/listing-view'
import {AuthorsView} from '@/components/authors-view'
import {StateMessage} from '@/components/state-message'
import {copy,localize} from '@/lib/i18n'
import {publicArticles} from '@/lib/content'
import {useDemoPipelineState} from './use-pipeline'
const loading=(locale:Locale)=><div className='state-message' role='status'>{copy[locale].loading}</div>
export function DemoListing(props:ListingProps){const state=useDemoPipelineState();if(!state)return loading(props.locale);const{locale,layer,categoryId,region,q,authorId}=props;const articles=publicArticles(Object.values(state.publications),locale).filter(a=>(!layer||a.layer===layer)&&(!categoryId||a.categoryId===categoryId)&&(!region||a.region===region)&&(!authorId||a.authorId===authorId)&&(!q||[localize(a.title,locale),localize(a.body,locale)].join(' ').toLocaleLowerCase().includes(q.toLocaleLowerCase()))).sort((a,b)=>(b.publishedAt||'').localeCompare(a.publishedAt||''));return <ArticleListingView {...props} articles={articles} categories={state.categories}/>}
export function DemoAuthors({locale}:{locale:Locale}){const state=useDemoPipelineState();return state?<AuthorsView locale={locale} authors={state.authors}/>:loading(locale)}
export function DemoAuthorReader({locale,id}:{locale:Locale;id:string}){const state=useDemoPipelineState();if(!state)return loading(locale);const author=state.authors.find(a=>a.id===id);if(!author)return <StateMessage locale={locale} title={copy[locale].notFound}/>;return <DemoListing locale={locale} title={author.name} description={localize(author.bio,locale)} introduction={<div className='author-avatar'>{author.avatarUrl?<img src={author.avatarUrl} alt=''/>:author.name.charAt(0)}</div>} authorId={id}/>}
