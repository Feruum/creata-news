import type { Metadata } from 'next'
import { getArticleCards,getCategories,getOpportunities } from '@/lib/api'
import { copy,isLocale } from '@/lib/i18n'
import { publicArticles } from '@/lib/content'
import { ClassicHomeScreen } from '@/features/news/components/classic-home-screen'
import {getNewspaperArchive} from '@/features/newspaper/public-api'
export async function generateMetadata({params}:{params:Promise<{locale:string}>}):Promise<Metadata>{const{locale}=await params;const l=isLocale(locale)?locale:'ru';return {title:copy[l].newsForAction,description:copy[l].platformDescription,alternates:{canonical:`/${l}`,languages:{ru:'/ru',kk:'/kk',en:'/en','x-default':'/ru'}}}}
export default async function Home({params}:{params:Promise<{locale:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';const[articles,categories,opportunities,issues]=await Promise.all([getArticleCards(locale,{limit:'24'}),getCategories(),getOpportunities(locale),getNewspaperArchive()]);return <ClassicHomeScreen articles={publicArticles(articles.data,locale)} categories={categories.data} opportunities={opportunities.data.filter(item=>item.status==='approved')} issues={issues.data} locale={locale} error={articles.error} opportunitiesError={opportunities.error}/>}
