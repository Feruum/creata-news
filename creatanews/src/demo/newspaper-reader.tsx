'use client'
import {useEffect,useState} from 'react'
import type {Locale,NewspaperPublication,OriginalIssueContent} from '@creata/contracts'
import {PublishedNewspaper} from '@/features/newspaper/published-issue'
import {OriginalNewspaperView} from '@/features/newspaper/original-view'
import {StateMessage} from '@/components/state-message'
import {copy} from '@/lib/i18n'
import {useDemoPipelineState} from './use-pipeline'
import {getDemoPipeline} from './pipeline'
export function DemoNewspaperReader({locale,slug,preview=false,previewId}:{locale:Locale;slug?:string;preview?:boolean;previewId?:string}){
 const state=useDemoPipelineState(),[draft,setDraft]=useState<NewspaperPublication>(),[error,setError]=useState(false)
 useEffect(()=>{if(!previewId)return;let active=true;void getDemoPipeline().then(engine=>engine.request<NewspaperPublication>('/admin/newspapers/'+encodeURIComponent(previewId)+'/preview')).then(issue=>{if(active)setDraft(issue)}).catch(()=>{if(active)setError(true)});return()=>{active=false}},[previewId])
 if(error)return <StateMessage locale={locale} error/>
 if(!state||previewId&&!draft)return <div className='state-message' role='status'>{copy[locale].loading}</div>
 let issue:NewspaperPublication|undefined=draft
 if(!previewId&&slug==='01'){
  const page=preview?state.pages.find(page=>page.slug==='newspaper-01'):state.pagePublications['newspaper-01']
  if(page)issue={...page.content as OriginalIssueContent,id:page.id,format:'original',revision:page.revision,publishedRevision:page.publishedRevision||page.revision,entries:[]}
 }else if(!previewId)issue=Object.values(state.newspaperPublications).find(issue=>issue.slug===slug)
 if(!issue)return <StateMessage locale={locale} title={copy[locale].notFound}/>
 return <>{(preview||previewId)&&<div className='preview-banner'>{copy[locale].previewNote}</div>}{issue.format==='original'?<OriginalNewspaperView locale={locale} issue={issue}/>:<PublishedNewspaper locale={locale} issue={issue}/>}</>
}
