import type { Metadata } from 'next'
import { Latest } from '@/components/latest'
import { copy,isLocale } from '@/lib/i18n'

export async function generateMetadata({params}:{params:Promise<{locale:string}>}):Promise<Metadata>{const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';return{title:copy[locale].latest,alternates:{canonical:`/${locale}/latest`,languages:{ru:'/ru/latest',kk:'/kk/latest',en:'/en/latest'}}}}
export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{section?:string;category?:string;region?:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';return <Latest locale={locale} {...await searchParams}/>}
