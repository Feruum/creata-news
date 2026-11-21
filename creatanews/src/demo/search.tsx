'use client'
import type {Locale} from '@/lib/models'
import {SearchView} from '@/components/search-view'
import {copy} from '@/lib/i18n'
import {useDemoPipelineState} from './use-pipeline'
export function DemoSearch({locale,q}:{locale:Locale;q:string}){
 const state=useDemoPipelineState()
 if(!state)return <div className='state-message' role='status'>{copy[locale].loading}</div>
 const needle=q.toLocaleLowerCase(),articles=q?Object.values(state.publications).filter(article=>[article.title[locale],article.lead[locale],article.body[locale],article.tags.join(' ')].join(' ').toLocaleLowerCase().includes(needle)):[]
 return <SearchView locale={locale} q={q} articles={articles} authors={state.authors} categories={state.categories} opportunities={state.opportunities.filter(item=>item.status==='approved')}/>
}
