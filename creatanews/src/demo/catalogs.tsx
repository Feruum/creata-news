'use client'
import type {Locale} from '@creata/contracts'
import {StateMessage} from '@/components/state-message'
import {OpportunityCatalog} from '@/features/opportunities/catalog-view'
import {OpportunityDetail} from '@/features/opportunities/detail-view'
import {validateDataset} from '@creata/graphics'
import {LaunchCatalog} from '@/features/launches/catalog-view'
import {LaunchDetail} from '@/features/launches/detail-view'
import {ChartsCatalog} from '@/features/charts/catalog'
import {GraphicStory} from '@/features/charts/story'
import {NewspaperArchive} from '@/features/newspaper/archive'
import {copy} from '@/lib/i18n'
import {useDemoPipelineState} from './use-pipeline'
import {demoNewspaperIssues} from './newspapers'
const loading=(locale:Locale)=><div className='state-message' role='status'>{copy[locale].loading}</div>
export function DemoOpportunities({locale,filters}:{locale:Locale;filters:{type?:string;country?:string;q?:string}}){const state=useDemoPipelineState();return state?<OpportunityCatalog locale={locale} filters={filters} data={state.opportunities}/>:loading(locale)}
export function DemoOpportunityReader({locale,id}:{locale:Locale;id:string}){const state=useDemoPipelineState();if(!state)return loading(locale);const item=state.opportunities.find(item=>item.id===id&&item.status==='approved');return item?<OpportunityDetail locale={locale} item={item}/>:<StateMessage locale={locale} title={copy[locale].notFound}/>}
export function DemoLaunches({locale,filters}:{locale:Locale;filters:Record<string,string|undefined>}){const state=useDemoPipelineState();return state?<LaunchCatalog locale={locale} filters={filters} data={state.launches.filter(item=>item.status==='published')} categories={state.categories}/>:loading(locale)}
export function DemoLaunchReader({locale,slug}:{locale:Locale;slug:string}){const state=useDemoPipelineState();if(!state)return loading(locale);const launch=state.launches.find(item=>item.slug===slug&&item.status==='published');return launch?<LaunchDetail locale={locale} launch={launch} others={state.launches.filter(item=>item.status==='published')} categories={state.categories}/>:<StateMessage locale={locale} title={copy[locale].notFound}/>}
export function DemoCharts({locale,query,type}:{locale:Locale;query:string;type:string}){const state=useDemoPipelineState();return state?<ChartsCatalog locale={locale} articles={Object.values(state.publications)} query={query} type={type}/>:loading(locale)}
export function DemoChartReader({locale,slug}:{locale:Locale;slug:string}){const state=useDemoPipelineState();if(!state)return loading(locale);const article=Object.values(state.publications).find(item=>item.slug===slug&&item.format==='graphic');return article?.graphic&&article.title[locale]?.trim()&&!validateDataset(article.graphic)?<GraphicStory locale={locale} article={article}/>:<StateMessage locale={locale} title={copy[locale].notFound}/>}
export function DemoNewspaperArchive({locale}:{locale:Locale}){
 const state=useDemoPipelineState();if(!state)return loading(locale)
 return <NewspaperArchive locale={locale} issues={demoNewspaperIssues(state)}/>
}
