import { getCategories } from '@/lib/api'
import { isLocale } from '@/lib/i18n'
import { Profile } from '@/components/profile'
export default async function Page({params}:{params:Promise<{locale:string}>}){const{locale:raw}=await params;const locale=isLocale(raw)?raw:'ru',categories=await getCategories();return <Profile locale={locale} categories={categories.data}/>}
