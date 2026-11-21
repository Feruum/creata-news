import { notFound } from 'next/navigation'
import { headers } from 'next/headers'
import type { Metadata } from 'next'
import type { ReactNode } from 'react'
import { getArticles,getCategories,getPopularArticles } from '@/lib/api'
import { publicArticles } from '@/lib/content'
import { isLocale } from '@/lib/i18n'
import { Shell } from '@/components/shell'
export async function generateMetadata({params}:{params:Promise<{locale:string}>}):Promise<Metadata>{const{locale}=await params;const path=(await headers()).get('x-creata-path')||`/${locale}`;const suffix=path.split('/').slice(2).join('/');const translated=(l:string)=>`/${l}${suffix?`/${suffix}`:''}`;const privatePage=/\/(profile|saved|preview|search)(\/|$)/.test(path);return {alternates:{canonical:path,languages:{ru:translated('ru'),kk:translated('kk'),en:translated('en')}},...(privatePage?{robots:{index:false,follow:false}}:{})}}
export default async function LocaleLayout({children,params}: {children:ReactNode;params:Promise<{locale:string}>}) {const{locale}=await params;if(!isLocale(locale))notFound();const[categories,articles,popular]=await Promise.all([getCategories(),getArticles(locale),getPopularArticles(locale)]);const regions=[...new Set(articles.data.map(article=>article.region).filter(Boolean))].slice(0,12);return <Shell locale={locale} categories={categories.data} regions={regions} popularArticles={publicArticles(popular.data,locale)}>{children}</Shell>}
