'use client'
import type {Locale} from '@/lib/models'
import {ArticleView} from '@/components/article-view'
import {StateMessage} from '@/components/state-message'
import {copy} from '@/lib/i18n'
import {useDemoPipelineState} from './use-pipeline'
export function DemoArticleReader({locale,slug,id,preview=false}:{locale:Locale;slug?:string;id?:string;preview?:boolean}){
 const state=useDemoPipelineState(),t=copy[locale]
 if(!state)return <div className='state-message' role='status'>{t.loading}</div>
 const published=Object.values(state.publications),article=(preview?state.articles:published).find(article=>article.id===id||article.slug===slug)
 if(!article)return <StateMessage locale={locale} title={t.articleUnavailable}/>
 return <ArticleView article={{...article,author:article.author||state.authors.find(author=>author.id===article.authorId)}} locale={locale} categories={state.categories} preview={preview} related={published.filter(item=>item.id!==article.id&&item.categoryId===article.categoryId).slice(0,5)} opportunities={state.opportunities.filter(item=>article.opportunityIds?.includes(item.id)&&item.status==='approved'&&new Date(item.deadline).getTime()>Date.now())}/>
}
