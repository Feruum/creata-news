import {getOpportunities} from '@/lib/api'
import {isLocale} from '@/lib/i18n'
import {OpportunityCatalog} from '@/features/opportunities/catalog-view'
import {DemoOpportunities} from '@/demo/catalogs'
import {demoMode} from '@/demo/config'
export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{type?:string;country?:string;q?:string}>}){const{locale:raw}=await params,locale=isLocale(raw)?raw:'ru',filters=await searchParams;if(demoMode)return <DemoOpportunities locale={locale} filters={filters}/>;const result=await getOpportunities(locale);return <OpportunityCatalog locale={locale} filters={filters} data={result.data} error={result.error}/>}