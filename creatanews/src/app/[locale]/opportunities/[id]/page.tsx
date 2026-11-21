import { notFound } from 'next/navigation'
import { serverApi,PublicApiError } from '@/lib/api'
import { copy,formatDate,isLocale,localize } from '@/lib/i18n'
import type { Opportunity } from '@/lib/models'
import { StateMessage } from '@/components/state-message'
import { regionLabel } from '@/lib/topics'
import { OpportunityCard } from '@/features/opportunities/opportunity-card'
import { opportunityType } from '@/features/opportunities/opportunity-copy'
import {OpportunityDetail} from '@/features/opportunities/detail-view'
import {demoMode} from '@/demo/config'
import {DemoOpportunityReader} from '@/demo/catalogs'
export default async function Page({params}:{params:Promise<{locale:string;id:string}>}){
  const{locale:raw,id}=await params,locale=isLocale(raw)?raw:'ru',t=copy[locale]
  if(demoMode)return <DemoOpportunityReader locale={locale} id={id}/>
  let item:Opportunity
  try{item=await serverApi<Opportunity>(`/public/opportunities/${encodeURIComponent(id)}`)}
  catch(error){if(error instanceof PublicApiError&&error.status===404)notFound();return <StateMessage locale={locale} error/>}
  if(item.status!=='approved')notFound()
  return <OpportunityDetail locale={locale} item={item}/>
}
