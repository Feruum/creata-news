import {adminDemo,adminPath} from './demo'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/shared/ui/components/table'
import { useState } from 'react'
import { Link, useParams } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { endpoints } from '@creata/contracts'
import { Activity, ArrowRight, Bookmark, FileText, Layers3, LayoutGrid, List, Plus, Search, Sparkles, Users } from 'lucide-react'
import { useAdmin, useCategories } from './hooks'
import { canManageEditorialCatalog, canViewAudit, emptyText, filterArticles, localized, statuses } from './domain'
import { Button, Badge, Empty, ErrorPanel, ExternalLink, Field, Loading, LocaleTabs, Modal, PageHead } from './ui'
import { formatDate, hasCopy, type CopyKey } from './i18n'
import type { Analytics, Article, AuditEntry, Author, Digest, List as DataList, Locale, Opportunity, Story } from './types'
import { EditorialSettings } from './settings-pages'
import { DeliveriesPanel } from './deliveries-panel'
import { CollectLinkDialog } from './collect-link-dialog'
import { GraphicsPage } from './graphics-page'
import { LaunchesPage } from './launches-page'
import { NewspapersPage } from './newspaper-pages'
import { BankDuplicates } from './bank-duplicates'
import {SiteContentPage} from './site-content-page'
import {MediaLibraryPage} from './media-library-page'
import {StageAnalytics} from './stage-analytics'
import { StoryFollowUpsPage } from './story-follow-ups-page'
import { MorningDigestSummary } from './morning-digest'

export function EditorialPage() {
 const {section}=useParams({strict:false}) as {section:string}
 if(section==='charts')return <GraphicsPage/>
 if(section==='launches')return <LaunchesPage/>
 if(section==='newspapers')return <NewspapersPage/>
 if(section==='content')return <SiteContentPage/>
 if(section==='media')return <MediaLibraryPage/>
 if(section==='story-follow-ups')return <StoryFollowUpsPage/>
 switch(section){case'today':return <ArticlesPage today/>;case'articles':return <ArticlesPage/>;case'bank':return <BankPage/>;case'opportunities':return <OpportunitiesPage/>;case'authors':return <AuthorsPage/>;case'digests':return <DigestsPage/>;case'analytics':return <AnalyticsPage/>;case'settings':return <EditorialSettings/>;case'audit':return <AuditPage/>;default:return <Empty title='404'/>}
}

function CreateArticle({onClose}:{onClose:()=>void}){
 const {api,t,locale,toast}=useAdmin();const client=useQueryClient();const [lang,setLang]=useState<Locale>(locale);const [title,setTitle]=useState(emptyText());const [layer,setLayer]=useState('world');const [categoryId,setCategoryId]=useState('');const categories=useCategories()
 const mutation=useMutation({mutationFn:()=>api<Article>('/admin/articles',{method:'POST',body:JSON.stringify({title,layer,categoryId,lead:emptyText(),body:emptyText(),summary:emptyText(),actions:emptyText(),tags:[],sources:[]})}),onSuccess:a=>{void client.invalidateQueries();toast(t('saved'));window.location.assign(adminPath(`/articles/${encodeURIComponent(a.id)}`))}})
 const close=()=>onClose()
 return <Modal title={t('newArticle')} onClose={close}><form className='form-stack' onSubmit={e=>{e.preventDefault();mutation.mutate()}}><LocaleTabs value={lang} onChange={setLang}/><Field label={t('title')}><input required value={title[lang]} onChange={e=>setTitle({...title,[lang]:e.target.value})}/></Field><div className='form-grid'><Field label={t('layer')}><select value={layer} onChange={e=>setLayer(e.target.value)}><option value='world'>{t('world')}</option><option value='eco'>{t('eco')}</option></select></Field><Field label={t('topic')}><select required value={categoryId} onChange={e=>setCategoryId(e.target.value)}><option value=''>{t('select')}</option>{categories.data?.items.map(c=><option value={c.id} key={c.id}>{localized(c.name,locale)}</option>)}</select></Field></div>{mutation.error&&<ErrorPanel error={mutation.error}/>}<div className='form-actions'><Button variant='secondary' type='button' onClick={close}>{t('cancel')}</Button><Button disabled={mutation.isPending}>{mutation.isPending?t('saving'):t('newArticle')}</Button></div></form></Modal>
}

export function ArticlesPage({today=false}:{today?:boolean}){
 const {api,t,locale}=useAdmin();const [create,setCreate]=useState(()=>new URLSearchParams(window.location.search).get('new')==='1'),[view,setView]=useState<'board'|'list'>(today?'board':'list')
 const [query,setQuery]=useState(()=>new URLSearchParams(window.location.search).get('q')||'');const [status,setStatus]=useState(''),[categoryId,setCategoryId]=useState(''),[authorId,setAuthorId]=useState('')
 const [language,setLanguage]=useState<Locale|''>(''),[from,setFrom]=useState(''),[to,setTo]=useState('');const articles=useQuery({queryKey:['articles'],queryFn:()=>api<DataList<Article>>('/admin/articles'),refetchInterval:30_000});const analytics=useQuery({queryKey:['analytics'],queryFn:()=>api<Analytics>('/admin/analytics')});const categories=useCategories();const authors=useQuery({queryKey:['authors'],queryFn:()=>api<DataList<Author>>('/admin/authors')})
 const items=filterArticles(articles.data?.items||[],{query,locale,status,categoryId,authorId,language,from,to}); const categoryName=(id:string)=>localized(categories.data?.items.find(c=>c.id===id)?.name,locale)
 return <><PageHead title={t(today?'today':'articles')} action={<Button onClick={()=>setCreate(true)}><Plus size={15}/>{t('newArticle')}</Button>}/>
 {today&&analytics.data&&<div className='stats-grid'>{[{label:'publishedMetric',value:analytics.data.published,icon:FileText},{label:'inReview',value:analytics.data.statuses?.review||0,icon:Activity},{label:'bankMetric',value:analytics.data.stories,icon:Layers3},{label:'subscribers',value:analytics.data.subscribers,icon:Users}].map(({label,value,icon:Icon})=><section className='stat-card' key={label}><header><span>{t(label as CopyKey)}</span><Icon size={15}/></header><div className='stat-value'>{value.toLocaleString(locale)}</div></section>)}</div>}
 {analytics.error&&today&&<ErrorPanel error={analytics.error} retry={()=>void analytics.refetch()}/>}
 {today&&!adminDemo&&<MorningDigestSummary/>}
 <div className='section-head'><h2>{t(today?'workflow':'articles')}{articles.data&&<span className='muted count-inline'> {articles.data.items.length}</span>}</h2><div className='segment'><button className={view==='board'?'active':''} onClick={()=>setView('board')} aria-pressed={view==='board'}><LayoutGrid size={13}/>{t('board')}</button><button className={view==='list'?'active':''} onClick={()=>setView('list')} aria-pressed={view==='list'}><List size={13}/>{t('list')}</button></div></div>
 <div className='toolbar'><label className='search-field'><Search size={16}/><input aria-label={t('search')} placeholder={t('searchArticles')} value={query} onChange={e=>setQuery(e.target.value)}/></label><select aria-label={t('status')} value={status} onChange={e=>setStatus(e.target.value)}><option value=''>{t('allStatuses')}</option>{statuses.map(s=><option key={s} value={s}>{t(s)}</option>)}</select><select aria-label={t('topic')} value={categoryId} onChange={e=>setCategoryId(e.target.value)}><option value=''>{t('allTopics')}</option>{categories.data?.items.map(c=><option key={c.id} value={c.id}>{localized(c.name,locale)}</option>)}</select><select aria-label={t('author')} value={authorId} onChange={e=>setAuthorId(e.target.value)}><option value=''>{t('allAuthors')}</option>{authors.data?.items.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select><select aria-label={t('contentLanguage')} value={language} onChange={e=>setLanguage(e.target.value as Locale|'')}><option value=''>{t('allLanguages')}</option><option value='ru'>RU</option><option value='kk'>ҚАЗ</option><option value='en'>EN</option></select><label className='date-filter'><span>{t('dateFrom')}</span><input type='date' value={from} onChange={e=>setFrom(e.target.value)}/></label><label className='date-filter'><span>{t('dateTo')}</span><input type='date' value={to} min={from} onChange={e=>setTo(e.target.value)}/></label></div>
 {articles.isPending?<Loading/>:articles.error?<ErrorPanel error={articles.error} retry={()=>void articles.refetch()}/>:!items.length?<div className='table-panel'><Empty title={query||status||categoryId||authorId||language||from||to?t('noResults'):undefined}>{(query||status||categoryId||authorId||language||from||to)?<Button variant='secondary' onClick={()=>{setQuery('');setStatus('');setCategoryId('');setAuthorId('');setLanguage('');setFrom('');setTo('')}}>{t('resetFilters')}</Button>:<Button variant='secondary' onClick={()=>setCreate(true)}><Plus size={14}/>{t('newArticle')}</Button>}</Empty></div>:view==='board'?<div className='kanban' aria-label={t('workflow')}>{statuses.filter(s=>!status||s===status).map(s=><section className='kanban-column' data-status={s} key={s}><header className='column-head'><span className='column-dot'/><span>{t(s)}</span><span className='count'>{items.filter(a=>a.status===s).length}</span></header><div className='column-cards'>{items.filter(a=>a.status===s).map(a=><Link to={`/articles/${encodeURIComponent(a.id)}`} className='story-card' key={a.id}><span className='card-kicker'>{categoryName(a.categoryId)||t(a.layer)}<span>·</span>{a.layer==='eco'?'CREATA':a.region}</span>{a.media?.url&&<img className='card-image' src={a.media.url} alt=''/>}<h3>{localized(a.title,locale)||t('untitled')}</h3>{localized(a.lead,locale)&&<p>{localized(a.lead,locale)}</p>}<div className='card-meta'><span className='card-author'><span className='mini-avatar'>{(a.author?.name||authors.data?.items.find(v=>v.id===a.authorId)?.name||'C').charAt(0)}</span><span>{a.author?.name||authors.data?.items.find(v=>v.id===a.authorId)?.name||t('team')}</span></span><span>{a.sources?.length||0} {t('sourceCount')}</span></div></Link>)}{!items.some(a=>a.status===s)&&<div className='column-empty'>—</div>}</div></section>)}</div>:<div className='table-panel'><Table><TableHeader><TableRow><TableHead>{t('title')}</TableHead><TableHead>{t('author')}</TableHead><TableHead>{t('topic')}</TableHead><TableHead>{t('status')}</TableHead><TableHead>{t('updated')}</TableHead></TableRow></TableHeader><TableBody>{items.map(a=><TableRow key={a.id}><TableCell><Link to={`/articles/${encodeURIComponent(a.id)}`} className='row-title'>{localized(a.title,locale)||t('untitled')}</Link><small>{t(a.layer)} · {t('revision')} {a.revision}</small></TableCell><TableCell>{a.author?.name||authors.data?.items.find(v=>v.id===a.authorId)?.name||'—'}</TableCell><TableCell>{categoryName(a.categoryId)||'—'}</TableCell><TableCell><Badge status={a.status}/></TableCell><TableCell className='muted'>{formatDate(a.updatedAt,locale,true)}</TableCell></TableRow>)}</TableBody></Table></div>}{create&&<CreateArticle onClose={()=>setCreate(false)}/>}</>
}

function BankPage(){
 const {api,t,locale,toast}=useAdmin();const client=useQueryClient();const [q,setQ]=useState(''),[lang,setLang]=useState(''),[selected,setSelected]=useState<Story|null>(null),[collectLink,setCollectLink]=useState(false)
 const stories=useQuery({queryKey:['bank'],queryFn:()=>api<DataList<Story>>('/admin/bank'),refetchInterval:20_000})
 const draft=useMutation({mutationFn:(id:string)=>api<Article>(`/admin/bank/${encodeURIComponent(id)}/draft`,{method:'POST'}),onSuccess:a=>{void client.invalidateQueries();toast(t('sourceQueued'));if(a.id)window.location.assign(adminPath(`/articles/${encodeURIComponent(a.id)}`))}})
 const items=(stories.data?.items||[]).filter(s=>(!lang||s.locale===lang)&&(!q||`${s.title} ${s.text} ${s.tags?.join(' ')}`.toLocaleLowerCase().includes(q.toLocaleLowerCase())))
 return <><PageHead title={t('bank')} action={<Button onClick={()=>setCollectLink(true)}><Plus size={14}/>{t('submitLink')}</Button>}/><BankDuplicates/><div className='toolbar'><label className='search-field'><Search size={16}/><input aria-label={t('search')} placeholder={`${t('search')}…`} value={q} onChange={e=>setQ(e.target.value)}/></label><select aria-label={t('language')} value={lang} onChange={e=>setLang(e.target.value)}><option value=''>{t('allLanguages')}</option><option>ru</option><option>kk</option><option>en</option></select></div>{draft.error&&<ErrorPanel error={draft.error}/>} {stories.isPending?<Loading/>:stories.error?<ErrorPanel error={stories.error} retry={()=>void stories.refetch()}/>:!items.length?<Empty/>:<div className='table-panel'><Table><TableHeader><TableRow><TableHead>{t('title')}</TableHead><TableHead>{t('score')}</TableHead><TableHead>{t('language')}</TableHead><TableHead>{t('status')}</TableHead><th/></TableRow></TableHeader><TableBody>{items.map(s=><TableRow key={s.id}><TableCell><button className='text-button row-title' onClick={()=>setSelected(s)}>{s.title}</button><small>{s.region||s.sourceType} · {formatDate(s.createdAt,locale)}</small></TableCell><TableCell><span className='score-pill'>{Number(s.score||0).toFixed(0)}</span></TableCell><TableCell>{s.locale==='kk'?'ҚАЗ':s.locale.toUpperCase()}</TableCell><TableCell><Badge status={s.status}/></TableCell><TableCell><Button variant='secondary' className='small' disabled={draft.isPending} onClick={()=>draft.mutate(s.id)}>{t('createDraft')}<ArrowRight size={13}/></Button></TableCell></TableRow>)}</TableBody></Table></div>}{collectLink&&<CollectLinkDialog onClose={()=>setCollectLink(false)}/>} {selected&&<Modal title={selected.title} onClose={()=>setSelected(null)} wide><div className='form-stack'><div className='callout'><strong>{t('reason')}</strong><p>{selected.reason||'—'}</p></div><p className='pre-wrap'>{selected.text}</p><ExternalLink href={selected.url}>{t('original')}</ExternalLink><div className='form-actions'><Button onClick={()=>draft.mutate(selected.id)} disabled={draft.isPending}>{t('createDraft')}</Button></div></div></Modal>}</>
}

function OpportunityForm({item,onClose}:{item?:Opportunity;onClose:()=>void}){
 const {api,t,locale,toast}=useAdmin();const client=useQueryClient();const [lang,setLang]=useState<Locale>(locale)
 const [value,setValue]=useState<Omit<Opportunity,'id'>>(()=>item?{...item,photo:item.photo||'',forWhom:{...emptyText(),...item.forWhom}}:{title:emptyText(),description:emptyText(),forWhom:emptyText(),photo:'',organization:'',type:'grant',url:'',country:'',deadline:'',status:'pending',categoryId:''})
 const cats=useCategories();const mutate=useMutation({mutationFn:()=>api(item?`/admin/opportunities/${encodeURIComponent(item.id)}`:'/admin/opportunities',{method:item?'PATCH':'POST',body:JSON.stringify(value)}),onSuccess:()=>{void client.invalidateQueries({queryKey:['opportunities']});toast(t('saved'));onClose()}})
 const update=(key:string,next:unknown)=>setValue(prev=>({...prev,[key]:next}));const localizedField=(key:'title'|'description'|'forWhom',text:string)=>update(key,{...(value[key]||emptyText()),[lang]:text})
 return <Modal title={t(item?'edit':'newOpportunity')} onClose={onClose}><form className='form-stack' onSubmit={e=>{e.preventDefault();mutate.mutate()}}>
  <LocaleTabs value={lang} onChange={setLang}/>
  <Field label={t('title')}><input required={value.status==='approved'} value={value.title[lang]} onChange={e=>localizedField('title',e.target.value)}/></Field>
  <Field label={t('description')}><textarea required={value.status==='approved'} value={value.description[lang]} onChange={e=>localizedField('description',e.target.value)}/></Field>
  <Field label={t('forWhom')}><textarea value={value.forWhom?.[lang]||''} onChange={e=>localizedField('forWhom',e.target.value)}/></Field>
  <div className='form-grid'><Field label={t('organization')}><input required={value.status==='approved'} value={value.organization} onChange={e=>update('organization',e.target.value)}/></Field><Field label={t('type')}><input required={value.status==='approved'} value={value.type} onChange={e=>update('type',e.target.value)}/></Field><Field label={t('country')}><input required={value.status==='approved'} value={value.country} onChange={e=>update('country',e.target.value)}/></Field><Field label={t('deadline')}><input required={value.status==='approved'} type='datetime-local' value={value.deadline?new Date(new Date(value.deadline).getTime()-new Date(value.deadline).getTimezoneOffset()*60000).toISOString().slice(0,16):''} onChange={e=>update('deadline',e.target.value?new Date(e.target.value).toISOString():'')}/></Field><Field label={t('topic')}><select value={value.categoryId} onChange={e=>update('categoryId',e.target.value)}><option value=''>{t('select')}</option>{cats.data?.items.map(c=><option key={c.id} value={c.id}>{localized(c.name,locale)}</option>)}</select></Field><Field label={t('status')}><select value={value.status} onChange={e=>update('status',e.target.value)}>{['pending','approved','rejected'].map(s=><option value={s} key={s}>{t(s as CopyKey)}</option>)}</select></Field></div>
  <Field label={t('url')}><input type='url' required={value.status==='approved'} value={value.url} onChange={e=>update('url',e.target.value)}/></Field>
  <Field label={t('avatar')}><input type='url' value={value.photo||''} onChange={e=>update('photo',e.target.value)}/></Field>
  {value.photo&&<div className='form-grid'><Field label={t('credit')}><input required={value.status==='approved'} value={value.photoRights?.credit||''} onChange={e=>update('photoRights',{...value.photoRights,credit:e.target.value})}/></Field><Field label={t('rights')}><input required={value.status==='approved'} value={value.photoRights?.rights||''} onChange={e=>update('photoRights',{...value.photoRights,rights:e.target.value})}/></Field></div>}
  {value.photo&&item?.needsRightsReview&&<Field label={locale==='ru'?'Права на фото проверены редакцией':locale==='kk'?'Фото құқықтарын редакция тексерді':'Photo rights reviewed by the newsroom'}><input type='checkbox' checked={value.needsRightsReview===false} onChange={e=>update('needsRightsReview',!e.target.checked)}/></Field>}
  <Field label={locale==='ru'?'Языки подборки':locale==='kk'?'Топтама тілдері':'Digest languages'}><div className='locale-tabs'>{(['ru','kk','en'] as Locale[]).map(l=><label key={l}><input type='checkbox' checked={(value.languages||['ru','kk','en']).includes(l)} onChange={e=>update('languages',e.target.checked?[...(value.languages||['ru','kk','en']),l]:(value.languages||['ru','kk','en']).filter(v=>v!==l))}/>{l==='kk'?'ҚАЗ':l.toUpperCase()}</label>)}</div></Field>
  {mutate.error&&<ErrorPanel error={mutate.error}/>}<div className='form-actions'><Button type='button' variant='secondary' onClick={onClose}>{t('cancel')}</Button><Button disabled={mutate.isPending}>{t('save')}</Button></div>
 </form></Modal>
}
function OpportunitiesPage(){
 const {api,t,locale,session}=useAdmin()
 const canManage=canManageEditorialCatalog(session?.user)
 const [editing,setEditing]=useState<Opportunity|null|undefined>(undefined)
 const [q,setQ]=useState('')
 const data=useQuery({queryKey:['opportunities'],queryFn:()=>api<DataList<Opportunity>>('/admin/opportunities')})
 const items=(data.data?.items||[]).filter(v=>`${localized(v.title,locale)} ${v.organization}`.toLowerCase().includes(q.toLowerCase()))
 return <>
  <PageHead title={t('opportunities')} action={canManage&&<Button onClick={()=>setEditing(null)}><Plus size={15}/>{t('newOpportunity')}</Button>}/>
  <div className='toolbar'><label className='search-field'><Search size={15}/><input value={q} aria-label={t('search')} placeholder={`${t('search')}…`} onChange={e=>setQ(e.target.value)}/></label></div>
  {data.isPending?<Loading/>:data.error?<ErrorPanel error={data.error} retry={()=>void data.refetch()}/>:!items.length?<Empty/>:
   <div className='table-panel'><Table>
    <TableHeader><TableRow><TableHead>{t('title')}</TableHead><TableHead>{t('organization')}</TableHead><TableHead>{t('deadline')}</TableHead><TableHead>{t('status')}</TableHead><th/></TableRow></TableHeader>
    <TableBody>{items.map(v=><TableRow key={v.id}>
     <TableCell>
      {canManage
       ? <button className='text-button row-title' onClick={()=>setEditing(v)}>{localized(v.title,locale)}</button>
       : <span className='row-title'>{localized(v.title,locale)}</span>}
      <small>{v.type} · {v.country}</small>
     </TableCell>
     <TableCell>{v.organization}</TableCell>
     <TableCell>{formatDate(v.deadline,locale,true)}</TableCell>
     <TableCell><Badge status={v.status}/></TableCell>
     <TableCell><ExternalLink href={v.url}>{t('open')}</ExternalLink></TableCell>
    </TableRow>)}</TableBody>
   </Table></div>}
  {canManage&&editing!==undefined&&<OpportunityForm item={editing||undefined} onClose={()=>setEditing(undefined)}/>}
 </>
}

function AuthorForm({item,onClose}:{item?:Author;onClose:()=>void}){
 const {api,t,locale,toast}=useAdmin();const client=useQueryClient();const [lang,setLang]=useState<Locale>(locale);const [value,setValue]=useState(()=>item?{...item}:{name:'',bio:emptyText(),avatarUrl:'',role:''});const mutate=useMutation({mutationFn:()=>api(item?`/admin/authors/${encodeURIComponent(item.id)}`:'/admin/authors',{method:item?'PATCH':'POST',body:JSON.stringify(value)}),onSuccess:()=>{void client.invalidateQueries({queryKey:['authors']});toast(t('saved'));onClose()}})
 return <Modal title={t(item?'edit':'newAuthor')} onClose={()=>onClose()}><form className='form-stack' onSubmit={e=>{e.preventDefault();mutate.mutate()}}><Field label={t('name')}><input required value={value.name} onChange={e=>setValue({...value,name:e.target.value})}/></Field><LocaleTabs value={lang} onChange={setLang}/><Field label={t('bio')}><textarea value={value.bio[lang]} onChange={e=>setValue({...value,bio:{...value.bio,[lang]:e.target.value}})}/></Field><Field label={t('avatar')}><input type='url' value={value.avatarUrl||''} onChange={e=>setValue({...value,avatarUrl:e.target.value})}/></Field><Field label={t('role')}><input value={value.role||''} onChange={e=>setValue({...value,role:e.target.value})}/></Field>{mutate.error&&<ErrorPanel error={mutate.error}/>}<div className='form-actions'><Button type='button' variant='secondary' onClick={onClose}>{t('cancel')}</Button><Button disabled={mutate.isPending}>{t('save')}</Button></div></form></Modal>
}
function AuthorsPage(){
 const {api,t,locale,session}=useAdmin()
 const canManage=canManageEditorialCatalog(session?.user)
 const [editing,setEditing]=useState<Author|null|undefined>()
 const data=useQuery({queryKey:['authors'],queryFn:()=>api<DataList<Author>>('/admin/authors')})
 return <>
  <PageHead title={t('authors')} action={canManage&&<Button onClick={()=>setEditing(null)}><Plus size={15}/>{t('newAuthor')}</Button>}/>
  {data.isPending?<Loading/>:data.error?<ErrorPanel error={data.error} retry={()=>void data.refetch()}/>:!data.data?.items.length?<Empty/>:
   <div className='author-grid'>{data.data.items.map(a=><article className='panel author-card' key={a.id}>
    {a.avatarUrl?<img className='author-avatar' src={a.avatarUrl} alt=''/>:<span className='author-avatar avatar'>{a.name.charAt(0)}</span>}
    <h2>{a.name}</h2>
    <span className='muted'>{a.role}</span>
    <p>{localized(a.bio,locale)}</p>
    {canManage&&<Button variant='secondary' className='small' onClick={()=>setEditing(a)}>{t('edit')}</Button>}
   </article>)}</div>}
  {canManage&&editing!==undefined&&<AuthorForm item={editing||undefined} onClose={()=>setEditing(undefined)}/>}
 </>
}

function DigestsPage(){
 const {api,t,locale,session,toast}=useAdmin()
 const client=useQueryClient()
 const [lang,setLang]=useState<Locale>(locale)
 const [selected,setSelected]=useState<Digest|null>(null)
 const [editionId]=useState(()=>new URLSearchParams(window.location.search).get('edition'))
 const [confirmSend,setConfirmSend]=useState(false)
 const roles=session?.user?.roles
 const canManage=roles?.some(role=>role==='editor'||role==='chief_editor')
 const canViewDeliveries=canManage||roles?.includes('admin')
 const data=useQuery({
  queryKey:['digests'],
  queryFn:()=>api<endpoints['GET /api/creata/v1/admin/digests']['response']>('/admin/digests'),
  refetchInterval:20_000,
 })
 const build=useMutation({mutationFn:()=>api<Digest>('/admin/digests/preview',{method:'POST',body:JSON.stringify({locale:lang,kind:'channel'})}),onSuccess:d=>{setSelected(d);void client.invalidateQueries({queryKey:['digests']})}})
 const action=useMutation({
  mutationFn:({id,action}:{id:string;action:'approve'|'send'})=>api<Digest>(`/admin/digests/${encodeURIComponent(id)}/${action}`,{method:'POST'}),
  onSuccess:(digest,request)=>{
   setSelected(digest)
   void client.invalidateQueries({queryKey:['digests']})
   void client.invalidateQueries({queryKey:['morningDigest']})
   if(request.action==='send') void client.invalidateQueries({queryKey:['deliveries']})
   toast(t(request.action==='approve'?'digestApproved':'digestQueued'))
   setConfirmSend(false)
  },
  onError:(_error,request)=>{
   if(request.action==='send') void data.refetch()
  },
 })
 const current=data.data?.items.find(d=>d.id===(selected?.id??editionId))??selected
 const canManageCurrent=canManage&&current?.kind==='channel'
 const deliveryStatus=data.data?.channelDeliveryStatus
 const deliveryBlock=deliveryStatus==='channel_missing'?t('digestChannelMissing'):
  deliveryStatus==='bot_missing'?t('digestBotMissing'):null
 const sendError=action.variables?.action==='send'?action.error:null
 const approveError=action.variables?.action==='approve'?action.error:null
 return <>
  <PageHead title={t('digests')} action={canManage&&<>
   <LocaleTabs value={lang} onChange={setLang}/>
   <Button disabled={build.isPending} onClick={()=>build.mutate()}><Plus size={14}/>{t('buildDigest')}</Button>
  </>}/>
  {(build.error||approveError)&&<ErrorPanel error={build.error||approveError}/>}
  <div className='digest-layout'>
   <div>{data.isPending?<Loading/>:data.error?<ErrorPanel error={data.error} retry={()=>void data.refetch()}/>:!data.data?.items.length?<Empty/>:
    <div className='digest-list'>{data.data.items.map(d=><button className={`digest-choice ${current?.id===d.id?'selected':''}`} onClick={()=>setSelected(d)} key={d.id}>
     <span><strong>{t(d.morningDate?'morningDigest':'channel')} · {d.locale==='kk'?'ҚАЗ':d.locale.toUpperCase()}</strong><small>{formatDate(d.createdAt,locale,true)} · {d.articles?.length||0} {t('articles').toLowerCase()}</small></span>
     <Badge status={d.status}/>
    </button>)}</div>}
   </div>
   <section className='panel'>{current?<>
    <div className='section-head'><h2>{t('preview')}</h2><Badge status={current.status}/></div>
    <div className='digest-preview'>{current.text||<>
     <strong>CR≡ATA NEWS</strong>
     {current.articles?.map(a=><div className='digest-item' key={a.id}><strong>{localized(a.title,current.locale)}</strong><p>{localized(a.summary||a.lead,current.locale)}</p></div>)}
     {current.outsideInterest&&<div className='digest-item'><strong>{localized(current.outsideInterest.title,current.locale)}</strong></div>}
     {current.opportunities?.map(o=><div className='digest-item' key={o.id}><strong>{localized(o.title,current.locale)}</strong><p>{o.organization} · {formatDate(o.deadline,locale)}</p></div>)}
    </>}</div>
    {!current.articles?.length&&!current.text&&<p className='muted'>{t('digestEmpty')}</p>}
    {canManageCurrent&&<div className='preview-actions'>
     {['draft','preview','new'].includes(current.status)&&<Button disabled={action.isPending||(!current.articles?.length&&!current.opportunities?.length&&!current.outsideInterest)} onClick={()=>action.mutate({id:current.id,action:'approve'})}>{t('approve')}</Button>}
     {current.status==='approved'&&<Button
      disabled={action.isPending||!!deliveryBlock}
      aria-describedby={deliveryBlock?'channel-delivery-unavailable':undefined}
      onClick={()=>{action.reset();setConfirmSend(true)}}
     >{t('send')}</Button>}
    </div>}
    {canManageCurrent&&current.status==='approved'&&deliveryBlock&&
     <p id='channel-delivery-unavailable' className='callout mt-3'>{deliveryBlock}</p>}
   </>:<Empty title={t('preview')}/>}</section>
  </div>
  {canViewDeliveries&&<DeliveriesPanel/>}
  {canManageCurrent&&confirmSend&&current&&<Modal title={t('send')} onClose={()=>setConfirmSend(false)}>
   <p className='panel-description'>{t('sendConfirm')}</p>
   {deliveryBlock?<p className='callout' role='alert'>{deliveryBlock}</p>:sendError&&<ErrorPanel error={sendError}/>}
   <div className='form-actions'>
    <Button variant='secondary' onClick={()=>setConfirmSend(false)}>{t('cancel')}</Button>
    <Button disabled={action.isPending||!!deliveryBlock||current.status!=='approved'} onClick={()=>action.mutate({id:current.id,action:'send'})}>{t('send')}</Button>
   </div>
  </Modal>}
 </>
}

function AnalyticsPage(){const {api,t,locale}=useAdmin();const data=useQuery({queryKey:['analytics'],queryFn:()=>api<Analytics>('/admin/analytics')});if(data.isPending)return <Loading/>;if(data.error)return <ErrorPanel error={data.error} retry={()=>void data.refetch()}/>;const a=data.data;const metrics:[CopyKey,number][]=[['publishedMetric',a.published],['bankMetric',a.stories],['subscribers',a.subscribers],['views',a.views],['saves',a.saves],['clicks',a.opportunityClicks],['aiSpend',a.aiCostUsd]];const max=Math.max(1,...Object.values(a.statuses||{}));return <><PageHead title={t('analytics')}/><div className='stats-grid'>{metrics.map(([key,v])=><section className='stat-card' key={key}><header><span>{t(key)}</span>{key==='saves'?<Bookmark size={15}/>:<Activity size={15}/>}</header><div className='stat-value'>{key==='aiSpend'?`$${v.toFixed(2)}`:v.toLocaleString(locale)}</div></section>)}</div><section className='panel'><h2>{t('workflow')}</h2><div className='bar-list'>{statuses.map(s=><div className='bar-row' key={s}><span>{t(s)}</span><span className='bar-track'><span style={{width:`${(a.statuses?.[s]||0)/max*100}%`}}/></span><strong>{a.statuses?.[s]||0}</strong></div>)}</div></section>{a.daily?.length>0&&<section className='panel section-spacing'><h2>{t('views')}</h2><div className='table-panel'><Table><TableBody>{a.daily.map((d,i)=><TableRow key={i}><TableCell>{formatDate(d.date,locale)}</TableCell><TableCell>{d.views??d.count??0}</TableCell></TableRow>)}</TableBody></Table></div></section>}<StageAnalytics/></>}
function AuditPage(){
 const {api,t,locale,session}=useAdmin()
 const allowed=canViewAudit(session?.user)
 const data=useQuery({queryKey:['audit'],queryFn:()=>api<DataList<AuditEntry>>('/admin/audit'),enabled:allowed})
 return <>
  <PageHead title={t('audit')}/>
  {!allowed?<p className='callout'>{t('forbidden')}</p>:
   data.isPending?<Loading/>:data.error?<ErrorPanel error={data.error} retry={()=>void data.refetch()}/>:!data.data?.items.length?<Empty/>:
   <div className='table-panel'><Table>
    <TableHeader><TableRow><TableHead>{t('created')}</TableHead><TableHead>{t('actor')}</TableHead><TableHead>{t('action')}</TableHead><TableHead>{t('target')}</TableHead></TableRow></TableHeader>
    <TableBody>{data.data.items.map(e=><TableRow key={e.id}>
     <TableCell>{formatDate(e.createdAt,locale,true)}</TableCell>
     <TableCell className='mono'>{e.actorId||'Hermes'}</TableCell>
     <TableCell>{hasCopy(e.action)?t(e.action):e.action}</TableCell>
     <TableCell className='mono'>{e.targetId}</TableCell>
    </TableRow>)}</TableBody>
   </Table></div>}
 </>
}
