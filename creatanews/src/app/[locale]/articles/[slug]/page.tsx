import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { ArticleView } from '@/components/article-view'
import { StateMessage } from '@/components/state-message'
import { getArticles,getAuthors,getCategories,getOpportunities,PublicApiError,serverApi } from '@/lib/api'
import { copy,isLocale,localize } from '@/lib/i18n'
import type { Article } from '@/lib/models'
import { publicArticles } from '@/lib/content'
import {demoMode} from '@/demo/config'
import {DemoArticleReader} from '@/demo/article-reader'
async function articleBySlug(slug:string){return serverApi<Article>(`/public/articles/${encodeURIComponent(slug)}`)}
export async function generateMetadata({params}:{params:Promise<{locale:string;slug:string}>}):Promise<Metadata>{const{locale:raw,slug}=await params;const locale=isLocale(raw)?raw:'ru';try{const article=await articleBySlug(slug);const title=localize(article.title,locale);return{title,description:localize(article.lead,locale),alternates:{canonical:`/${locale}/articles/${slug}`,languages:{ru:`/ru/articles/${slug}`,kk:`/kk/articles/${slug}`,en:`/en/articles/${slug}`}},openGraph:{type:'article',title,description:localize(article.lead,locale),publishedTime:article.publishedAt,images:article.media?.url?[article.media.url]:[]}}}catch{return{title:copy[locale].notFound,robots:{index:false,follow:false}}}}
export default async function Page({params}:{params:Promise<{locale:string;slug:string}>}){const{locale:raw,slug}=await params;const locale=isLocale(raw)?raw:'ru';if(demoMode)return <DemoArticleReader locale={locale} slug={slug}/>;let article:Article;try{article=await articleBySlug(slug)}catch(error){if(error instanceof PublicApiError&&error.status===404)notFound();return <StateMessage locale={locale} error/>}if(article.status!=='published')notFound();const[categories,related,authors,opportunities]=await Promise.all([getCategories(),getArticles(locale,{category:article.categoryId}),getAuthors(),getOpportunities(locale)]);article={...article,author:article.author||authors.data.find(item=>item.id===article.authorId)};return <ArticleView article={article} locale={locale} categories={categories.data} opportunities={opportunities.data.filter(o=>article.opportunityIds?.includes(o.id)&&o.status==='approved'&&new Date(o.deadline).valueOf()>Date.now())} related={publicArticles(related.data,locale).filter(item=>item.id!==article.id).slice(0,5)}/>}
