import type {Metadata} from 'next'
import {notFound} from 'next/navigation'
import {GraphicStory} from '@/features/charts/story'
import {PublicApiError,serverApi} from '@/lib/api'
import {isLocale,localize} from '@/lib/i18n'
import {StateMessage} from '@/components/state-message'
import {validateDataset} from '@creata/graphics'
import type {Article} from '@creata/contracts'
import {demoMode} from '@/demo/config'
import {DemoChartReader} from '@/demo/catalogs'
const get=(slug:string)=>serverApi<Article>(`/public/articles/${encodeURIComponent(slug)}`)
export async function generateMetadata({params}:{params:Promise<{locale:string;slug:string}>}):Promise<Metadata>{const{locale,slug}=await params,l=isLocale(locale)?locale:'ru';try{const a=await get(slug);return{title:localize(a.title,l),description:localize(a.lead,l),alternates:{canonical:`/${l}/charts/${slug}`,languages:{ru:`/ru/charts/${slug}`,kk:`/kk/charts/${slug}`,en:`/en/charts/${slug}`}},openGraph:{type:'article',title:localize(a.title,l),description:localize(a.lead,l),publishedTime:a.publishedAt}}}catch{return{robots:{index:false,follow:false}}}}
export default async function Page({params}:{params:Promise<{locale:string;slug:string}>}){const{locale,slug}=await params,l=isLocale(locale)?locale:'ru';if(demoMode)return <DemoChartReader locale={l} slug={slug}/>;let a:Article;try{a=await get(slug)}catch(e){if(e instanceof PublicApiError&&e.status===404)notFound();return <StateMessage locale={l} error/>}if(a.format!=='graphic'||!a.graphic||a.status!=='published'||!a.title[l]?.trim()||validateDataset(a.graphic))notFound();return <GraphicStory article={a} locale={l}/>}
