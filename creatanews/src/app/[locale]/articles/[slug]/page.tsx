import type { Metadata } from 'next'
import { headers } from 'next/headers'
import { notFound } from 'next/navigation'
import { ArticleView } from '@/components/article-view'
import { StateMessage } from '@/components/state-message'
import { getArticleCards,getCategories,getOpportunities,PublicApiError,serverApi } from '@/lib/api'
import { copy,isLocale,localize } from '@/lib/i18n'
import type { Article } from '@/lib/models'
import { publicArticles } from '@/lib/content'
import {demoMode} from '@/demo/config'
import {DemoArticleReader} from '@/demo/article-reader'

type ArticleParams = Promise<{locale:string;slug:string}>

async function articleRoute(params:ArticleParams) {
 const {locale:raw,slug}=await params
 const locale=isLocale(raw)?raw:'ru'
 const pathname=(await headers()).get('x-creata-path')
 const prefix=`/${locale}/articles/`
 // Metadata and page rendering can receive different encodings of the same route param.
 const segment=pathname?.startsWith(prefix)?pathname.slice(prefix.length):''
 if(segment&&!segment.includes('/')){
  try{return {locale,slug:decodeURIComponent(segment)}}
  catch{return {locale,slug}}
 }
 return {locale,slug}
}

async function articleBySlug(slug:string){return serverApi<Article>(`/public/articles/${encodeURIComponent(slug)}`)}
export async function generateMetadata({params}:{params:ArticleParams}):Promise<Metadata>{
 const {locale,slug}=await articleRoute(params)
 try{
  const article=await articleBySlug(slug)
  const title=localize(article.title,locale)
  const path=`/articles/${encodeURIComponent(article.slug)}`
  return{
   title,
   description:localize(article.lead,locale),
   alternates:{canonical:`/${locale}${path}`,languages:{ru:`/ru${path}`,kk:`/kk${path}`,en:`/en${path}`}},
   openGraph:{type:'article',title,description:localize(article.lead,locale),publishedTime:article.publishedAt,images:article.media?.url?[article.media.url]:[]},
  }
 }catch{return{title:copy[locale].notFound,robots:{index:false,follow:false}}}
}
export default async function Page({params}:{params:ArticleParams}) {
 const {locale,slug} = await articleRoute(params)
 if (demoMode) return <DemoArticleReader locale={locale} slug={slug}/>
 let article: Article
 try {
   article = await articleBySlug(slug)
 } catch (error) {
   if (error instanceof PublicApiError && error.status === 404) notFound()
   return <StateMessage locale={locale} error/>
 }
 if (article.status !== 'published') notFound()
 const [categories, related, opportunities] = await Promise.all([
   getCategories(),
   getArticleCards(locale, {category: article.categoryId, limit: '6'}),
   getOpportunities(locale, article.opportunityIds ?? []),
 ])
 return <ArticleView article={article} locale={locale} categories={categories.data}
   opportunities={opportunities.data.filter(o=>article.opportunityIds?.includes(o.id)&&o.status==='approved'&&new Date(o.deadline).valueOf()>Date.now())}
   related={publicArticles(related.data,locale).filter(item=>item.id!==article.id).slice(0,5)}/>
}
