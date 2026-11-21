import { getCategories } from '@/lib/api'
import { copy,isLocale } from '@/lib/i18n'
import { Saved } from '@/components/saved'
export default async function Page({params}:{params:Promise<{locale:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru',categories=await getCategories();return <><div className='page-heading'><span className='eyebrow'>{copy[locale].personal}</span><h1>{copy[locale].saved}</h1></div><Saved locale={locale} categories={categories.data}/></>}
