import { cookies } from 'next/headers'
import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import { apiOrigin,getAuthors,getCategories } from '@/lib/api'
import { isLocale } from '@/lib/i18n'
import type { Article } from '@/lib/models'
import { ArticleView } from '@/components/article-view'
import { StateMessage } from '@/components/state-message'
import {demoMode} from '@/demo/config'
import {DemoArticleReader} from '@/demo/article-reader'
export const metadata:Metadata={robots:{index:false,follow:false},title:'Editorial preview'}
export default async function Page({params}:{params:Promise<{locale:string;id:string}>}){const{locale:raw,id}=await params;const locale=isLocale(raw)?raw:'ru';if(demoMode)return <DemoArticleReader locale={locale} id={id} preview/>;try{const response=await fetch(`${apiOrigin()}/api/creata/v1/admin/articles/${encodeURIComponent(id)}`,{headers:{cookie:(await cookies()).toString()},cache:'no-store'});if([401,403,404].includes(response.status))notFound();if(!response.ok)return <StateMessage locale={locale} error/>;const article:Article=await response.json();const[categories,authors]=await Promise.all([getCategories(),getAuthors()]);return <ArticleView article={{...article,author:article.author||authors.data.find(item=>item.id===article.authorId)}} locale={locale} categories={categories.data} preview/>}catch(error){if(error instanceof Error&&'digest'in error)throw error;return <StateMessage locale={locale} error/>}}
