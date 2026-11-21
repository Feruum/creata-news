import { permanentRedirect } from 'next/navigation'
import { isLocale,localePath } from '@/lib/i18n'
import { buildQuery } from '@/lib/content'
export default async function Page({params,searchParams}:{params:Promise<{locale:string}>;searchParams:Promise<{category?:string;region?:string;section?:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru';permanentRedirect(localePath(locale,'/latest')+buildQuery(await searchParams))}
