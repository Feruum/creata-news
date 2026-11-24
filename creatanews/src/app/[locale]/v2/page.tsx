import type { Metadata } from 'next'
import { getArticleCards,getCategories } from '@/lib/api'
import { copy,isLocale } from '@/lib/i18n'
import { publicArticles } from '@/lib/content'
import { HomeScreen } from '@/features/news/components/home-screen'
import {getNewspaperArchive} from '@/features/newspaper/public-api'
export async function generateMetadata({params}:{params:Promise<{locale:string}>}):Promise<Metadata>{const{locale}=await params;const l=isLocale(locale)?locale:'ru';return {title:copy[l].newsForAction,description:copy[l].platformDescription,alternates:{canonical:`/${l}/v2`,languages:{ru:'/ru/v2',kk:'/kk/v2',en:'/en/v2','x-default':'/ru/v2'}}}}
export default async function HomeV2({params}:{params:Promise<{locale:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';const[articles,categories,issues]=await Promise.all([getArticleCards(locale,{limit:'24'}),getCategories(),getNewspaperArchive()]);return <HomeScreen articles={publicArticles(articles.data,locale)} categories={categories.data} issues={issues.data} locale={locale} error={articles.error}/>}
