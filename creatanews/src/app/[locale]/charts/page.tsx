import type {Metadata} from 'next'
import {ChartsCatalog} from '@/features/charts/catalog'
import {getArticles} from '@/lib/api'
import {isLocale} from '@/lib/i18n'
import {StateMessage} from '@/components/state-message'
import {graphicCopy} from '@creata/graphics'
import {demoMode} from '@/demo/config'
import {DemoCharts} from '@/demo/catalogs'
export async function generateMetadata({params}:{params:Promise<{locale:string}>}):Promise<Metadata>{const{locale}=await params;const l=isLocale(locale)?locale:'ru';return{title:graphicCopy[l].title,description:graphicCopy[l].intro,alternates:{canonical:`/${l}/charts`,languages:{ru:'/ru/charts',kk:'/kk/charts',en:'/en/charts'}}}}
export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{q?:string;type?:string}>}){const{locale}=await params,l=isLocale(locale)?locale:'ru',filters=await searchParams;if(demoMode)return <DemoCharts locale={l} query={filters.q||''} type={filters.type||''}/>;const data=await getArticles(l,{format:'graphic'});if(data.error)return <StateMessage locale={l} error/>;return <ChartsCatalog articles={data.data} locale={l} query={filters.q||''} type={filters.type||''}/>}
