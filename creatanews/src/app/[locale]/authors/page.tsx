import {getAuthors} from '@/lib/api'
import {isLocale} from '@/lib/i18n'
import {AuthorsView} from '@/components/authors-view'
import {demoMode} from '@/demo/config'
import {DemoAuthors} from '@/demo/listings'
export default async function Page({params}:{params:Promise<{locale:string}>}){const{locale:raw}=await params,locale=isLocale(raw)?raw:'ru';if(demoMode)return <DemoAuthors locale={locale}/>;const authors=await getAuthors();return <AuthorsView locale={locale} authors={authors.data} error={authors.error}/>}