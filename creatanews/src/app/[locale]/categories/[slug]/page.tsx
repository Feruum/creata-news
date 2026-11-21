import { notFound } from 'next/navigation'
import type { Metadata } from 'next'
import { ArticleListing } from '@/components/listing'
import { getCategories } from '@/lib/api'
import { copy,isLocale,localize } from '@/lib/i18n'
export async function generateMetadata({params}:{params:Promise<{locale:string;slug:string}>}):Promise<Metadata>{const{locale:raw,slug}=await params;const locale=isLocale(raw)?raw:'ru',categories=await getCategories();const category=categories.data.find(item=>item.slug===slug||item.id===slug);return{title:category?localize(category.name,locale):copy[locale].topics}}
export default async function Page({params,searchParams}:{params:Promise<{locale:string;slug:string}>;searchParams:Promise<{region?:string}>}){const{locale:raw,slug}=await params;const locale=isLocale(raw)?raw:'ru',categories=await getCategories();const category=categories.data.find(item=>item.slug===slug||item.id===slug);if(!category&&!categories.error)notFound();const{region}=await searchParams;return <ArticleListing locale={locale} title={category?localize(category.name,locale):copy[locale].topics} categoryId={category?.id} region={region}/>}
