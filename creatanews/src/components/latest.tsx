import type {Locale} from '@/lib/models'
import {getArticles,getCategories} from '@/lib/api'
import {demoMode} from '@/demo/config'
import {DemoLatest} from '@/demo/latest'
import {LatestView} from './latest-view'
export async function Latest(props:{locale:Locale;section?:string;category?:string;region?:string}){
 const[articles,categories]=await Promise.all([getArticles(props.locale),getCategories()])
 const Component=demoMode?DemoLatest:LatestView
 return <Component {...props} items={articles.data} categories={categories.data} error={articles.error}/>
}
