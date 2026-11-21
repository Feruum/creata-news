import { useQuery } from '@tanstack/react-query'
import { useAdmin } from './hooks'
import { localized } from './domain'
import { ErrorPanel, ExternalLink } from './ui'
import type { Job, List, Opportunity } from './types'

type Brief={facts?:Array<{claim:string;sourceUrls:string[];numbers?:Array<{value:string;sourceUrl:string;quote:string}>;dates?:Array<{value:string;sourceUrl:string;quote:string}>}>;conflicts?:string[];singleSourceClaims?:string[];unavailableSources?:string[]}
type PipelineJob=Job&{progress?:{phase:string;status:string};checkpoints?:Record<string,{status:string;error?:string;output?:Brief}>}
const copy={ru:{research:'Исследование',draft:'Черновик',review:'Проверка',succeeded:'Готово',running:'В работе',failed:'Ошибка',pending:'Ожидает',brief:'Исследовательская справка',conflicts:'Расхождения источников',single:'Один источник',unavailable:'Текст источника недоступен',opportunities:'Связанные возможности',none:'Возможности не связаны',progress:'Этапы Hermes'},kk:{research:'Зерттеу',draft:'Қолжазба',review:'Тексеру',succeeded:'Дайын',running:'Орындалуда',failed:'Қате',pending:'Күтуде',brief:'Зерттеу анықтамасы',conflicts:'Дереккөздер қайшылығы',single:'Бір дереккөз',unavailable:'Дереккөз мәтіні қолжетімсіз',opportunities:'Байланысты мүмкіндіктер',none:'Мүмкіндіктер байланыстырылмаған',progress:'Hermes кезеңдері'},en:{research:'Research',draft:'Draft',review:'Review',succeeded:'Complete',running:'In progress',failed:'Failed',pending:'Pending',brief:'Research brief',conflicts:'Source conflicts',single:'Single-source claims',unavailable:'Unavailable source text',opportunities:'Linked opportunities',none:'No linked opportunities',progress:'Hermes phases'}}

export function PipelineProgress({job,brief=false}:{job:PipelineJob;brief?:boolean}){
 const {locale}=useAdmin();const c=copy[locale];const research=job.checkpoints?.research?.output
 if(!['draft','rework'].includes(job.kind))return null
 return <div className='form-stack'><div aria-label={c.progress} className='job-detail-meta'>{(['research','draft','review'] as const).map(phase=>{const state=job.checkpoints?.[phase];return <span key={phase}><strong>{c[phase]}</strong>: {state?.status==='succeeded'?c.succeeded:state?.status==='running'?c.running:state?.status==='failed'?c.failed:c.pending}{state?.error&&<small className='source-error'>{state.error}</small>}</span>})}</div>{brief&&research&&<ResearchBrief value={research}/>}</div>
}

function ResearchBrief({value}:{value:Brief}){
 const {locale}=useAdmin();const c=copy[locale]
 return <section className='form-stack'><h3>{c.brief}</h3>{value.facts?.map((fact,i)=><div className='reference' key={i}><p>{fact.claim}</p>{fact.sourceUrls.map(url=><p key={url}><ExternalLink href={url}>{url}</ExternalLink></p>)}{[...(fact.numbers||[]),...(fact.dates||[])].map((proof,j)=><blockquote key={j}><strong>{proof.value}</strong> — {proof.quote}</blockquote>)}</div>)}{(['conflicts','singleSourceClaims','unavailableSources'] as const).map(field=>value[field]?.length?<div className='callout' key={field}><strong>{c[field==='conflicts'?'conflicts':field==='singleSourceClaims'?'single':'unavailable']}</strong>{value[field]?.map((text,i)=><p key={i}>{text}</p>)}</div>:null)}</section>
}

export function ArticleResearch({articleId,opportunityIds=[],researchBrief}:{articleId:string;opportunityIds?:string[];researchBrief?:Brief}){
 const {api,locale}=useAdmin();const c=copy[locale]
 const jobs=useQuery({queryKey:['article-research',articleId],queryFn:()=>api<List<PipelineJob>>(`/admin/articles/${encodeURIComponent(articleId)}/research`),refetchInterval:5000})
 const opportunities=useQuery({queryKey:['opportunities'],queryFn:()=>api<List<Opportunity>>('/admin/opportunities'),enabled:opportunityIds.length>0})
 const job=jobs.data?.items.find(j=>['draft','rework'].includes(j.kind));const linked=opportunities.data?.items.filter(o=>opportunityIds.includes(o.id))||[]
 return <><section className='panel'><h2>{c.progress}</h2>{jobs.error&&<ErrorPanel error={jobs.error}/>} {job?<PipelineProgress job={job} brief/>:researchBrief?<ResearchBrief value={researchBrief}/>:<p className='muted'>—</p>}</section><section className='panel'><h2>{c.opportunities}</h2>{opportunities.error&&<ErrorPanel error={opportunities.error}/>} {linked.length?linked.map(o=><div className='reference' key={o.id}><strong>{localized(o.title,locale)}</strong><p>{o.organization}</p><ExternalLink href={o.url}>{o.url}</ExternalLink></div>):<p className='panel-description'>{c.none}</p>}</section></>
}
