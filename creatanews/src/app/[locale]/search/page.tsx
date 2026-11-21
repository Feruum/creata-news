import {SearchView} from '@/components/search-view'
import {demoMode} from '@/demo/config'
import {DemoSearch} from '@/demo/search'
import Link from 'next/link'
import { ArrowUpRight,Search } from 'lucide-react'
import { getArticles,getAuthors,getCategories,getOpportunities } from '@/lib/api'
import { copy,isLocale,localePath,localize } from '@/lib/i18n'
import { publicArticles } from '@/lib/content'
import { matchSearchGroups,searchAvailability } from '@/lib/search-results'
import { ArticleCard,OpportunityCard } from '@/components/cards'
import { StateMessage } from '@/components/state-message'

export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{q?:string}>}){
  const{locale:raw}=await params
  const locale=isLocale(raw)?raw:'ru',t=copy[locale],{q:rawQuery}=await searchParams,q=(rawQuery||'').trim().slice(0,200)
  if(demoMode)return <DemoSearch locale={locale} q={q}/>
  const[articleResult,authors,categories,opportunities]=await Promise.all([getArticles(locale,{q}),getAuthors(),getCategories(),getOpportunities(locale)])
  const articles=q?publicArticles(articleResult.data,locale):[]
  return <SearchView locale={locale} q={q} articles={articles} authors={authors.data} categories={categories.data} opportunities={opportunities.data} errors={[articleResult.error,authors.error,categories.error,opportunities.error]}/>
}
