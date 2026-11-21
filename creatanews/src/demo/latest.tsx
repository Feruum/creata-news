'use client'
import type {Article,Category,Locale} from '@/lib/models'
import {LatestView} from '@/components/latest-view'
import {useDemoArticles} from './use-pipeline'
export function DemoLatest(props:{locale:Locale;section?:string;category?:string;region?:string;items:Article[];categories:Category[];error?:boolean}){
 const articles=useDemoArticles(props.items)
 return <LatestView {...props} items={articles}/>
}
