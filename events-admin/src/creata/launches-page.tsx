import {adminDemo,applicationUrls} from './demo'
import { useState } from 'react'
import { useMutation,useQuery,useQueryClient } from '@tanstack/react-query'
import { Plus,Search } from 'lucide-react'
import type { Launch,List,Locale } from '@creata/contracts'
import { Table,TableBody,TableCell,TableHead,TableHeader,TableRow } from '@/shared/ui/components/table'
import { useAdmin,useCategories } from './hooks'
import { emptyText,localized } from './domain'
import { formatDate } from './i18n'
import { Badge,Button,Empty,ErrorPanel,ExternalLink,Field,Loading,LocaleTabs,Modal,PageHead } from './ui'
import { LaunchReviewQueue } from './launch-review-queue'

function LaunchForm({item,onClose}:{item?:Launch;onClose:()=>void}) {
  const {api,t,locale,toast}=useAdmin(),client=useQueryClient(),categories=useCategories()
  const [lang,setLang]=useState<Locale>(locale)
  const [tags,setTags]=useState(item?.tags?.join(', ')||'')
  const [value,setValue]=useState<Omit<Launch,'id'|'revision'|'createdAt'|'updatedAt'>>(()=>item?{...item}:{slug:'',name:'',tagline:emptyText(),description:emptyText(),team:emptyText(),url:'',launchDate:'',country:'',categoryId:'startups',photo:'',status:'draft'})
  const mutation=useMutation({mutationFn:()=>api<Launch>(item?`/admin/launches/${encodeURIComponent(item.id)}`:'/admin/launches',{method:item?'PATCH':'POST',body:JSON.stringify({...value,tags:[...new Set(tags.split(',').map(tag=>tag.trim()).filter(Boolean))],...(item?{expectedRevision:item.revision}:{})})}),onSuccess:()=>{void client.invalidateQueries({queryKey:['launches']});toast(t('saved'));onClose()}})
  const update=<K extends keyof typeof value>(key:K,next:typeof value[K])=>setValue(current=>({...current,[key]:next}))
  return <Modal title={t(item?'edit':'newLaunch')} onClose={onClose} disablePointerDismissal><form className='form-stack' onSubmit={event=>{event.preventDefault();mutation.mutate()}}>
    <div className='form-grid'>
      <Field label={t('launchName')}><input required maxLength={200} value={value.name} onChange={e=>update('name',e.target.value)}/></Field>
      <Field label={t('launchSlug')}><input required pattern='[a-z0-9]+(-[a-z0-9]+)*' maxLength={100} readOnly={!!item} value={value.slug} onChange={e=>update('slug',e.target.value)}/></Field>
    </div>
    <LocaleTabs value={lang} onChange={setLang}/>
    <Field label={t('launchTagline')}><input maxLength={20000} value={value.tagline[lang]} onChange={e=>update('tagline',{...value.tagline,[lang]:e.target.value})}/></Field>
    <Field label={t('launchTags')}><input value={tags} onChange={e=>setTags(e.target.value)}/></Field>
    <Field label={t('description')}><textarea maxLength={20000} value={value.description[lang]} onChange={e=>update('description',{...value.description,[lang]:e.target.value})}/></Field>
    <Field label={t('launchTeam')}><textarea maxLength={20000} value={value.team[lang]} onChange={e=>update('team',{...value.team,[lang]:e.target.value})}/></Field>
    <div className='form-grid'>
      <Field label={t('launchDate')}><input required type='date' value={value.launchDate} onChange={e=>update('launchDate',e.target.value)}/></Field>
      <Field label={t('country')}><input maxLength={100} value={value.country} onChange={e=>update('country',e.target.value)}/></Field>
      <Field label={t('topic')}><select value={value.categoryId} onChange={e=>update('categoryId',e.target.value)}><option value=''>{t('select')}</option>{categories.data?.items.map(category=><option key={category.id} value={category.id}>{localized(category.name,locale)}</option>)}</select></Field>
      <Field label={t('status')}><select value={value.status} onChange={e=>update('status',e.target.value as Launch['status'])}><option value='draft'>{t('launchDraft')}</option><option value='published'>{t('published')}</option><option value='archived'>{t('launchArchived')}</option></select></Field>
    </div>
    <Field label={t('launchWebsite')}><input required type='url' value={value.url} onChange={e=>update('url',e.target.value)}/></Field>
    <Field label={t('launchPhoto')}><input value={value.photo||''} onChange={e=>update('photo',e.target.value)}/></Field>
    <Field label={t('launchLogo')}><input value={value.logoUrl||''} onChange={e=>update('logoUrl',e.target.value)}/></Field>
    {mutation.error&&<ErrorPanel error={mutation.error}/>}
    <div className='form-actions'><Button type='button' variant='secondary' onClick={onClose}>{t('cancel')}</Button><Button disabled={mutation.isPending}>{mutation.isPending?t('saving'):t('save')}</Button></div>
  </form></Modal>
}

export function LaunchesPage() {
  const {api,t,locale,session}=useAdmin()
  const [editing,setEditing]=useState<Launch|null|undefined>(),[q,setQ]=useState(''),[status,setStatus]=useState('')
  const data=useQuery({queryKey:['launches'],queryFn:()=>api<List<Launch>>('/admin/launches')})
  const canManage=session?.user?.roles.some(role=>['admin','chief_editor','editor'].includes(role))
  const items=(data.data?.items||[]).filter(item=>(!status||item.status===status)&&`${item.name} ${localized(item.tagline,locale)}`.toLowerCase().includes(q.toLowerCase()))
  return <>
    <PageHead title={t('launches')} action={canManage&&<Button onClick={()=>setEditing(null)}><Plus size={15}/>{t('newLaunch')}</Button>}/>
    {canManage&&!adminDemo&&<LaunchReviewQueue/>}
    <div className='toolbar'><label className='search-field'><Search size={15}/><input aria-label={t('search')} value={q} placeholder={t('search')} onChange={e=>setQ(e.target.value)}/></label><select aria-label={t('status')} value={status} onChange={e=>setStatus(e.target.value)}><option value=''>{t('allStatuses')}</option><option value='draft'>{t('launchDraft')}</option><option value='published'>{t('published')}</option><option value='archived'>{t('launchArchived')}</option></select></div>
    {data.isPending?<Loading/>:data.error?<ErrorPanel error={data.error} retry={()=>void data.refetch()}/>:!items.length?<Empty/>:<div className='table-panel'><Table><TableHeader><TableRow><TableHead>{t('name')}</TableHead><TableHead>{t('country')}</TableHead><TableHead>{t('launchDate')}</TableHead><TableHead>{t('status')}</TableHead><TableHead>{t('open')}</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.id}><TableCell>{canManage?<button className='text-button row-title' onClick={()=>setEditing(item)}>{item.name}</button>:item.name}<small>{localized(item.tagline,locale)}</small></TableCell><TableCell>{item.country||'—'}</TableCell><TableCell>{formatDate(`${item.launchDate}T12:00:00Z`,locale)}</TableCell><TableCell>{item.status==='published'?<Badge status='published'/>:<span className={`badge ${item.status}`}>{t(item.status==='draft'?'launchDraft':'launchArchived')}</span>}</TableCell><TableCell>{item.status==='published'&&<ExternalLink href={`${applicationUrls.web}/${locale}/launches/${item.slug}`}>{t('open')}</ExternalLink>}</TableCell></TableRow>)}</TableBody></Table></div>}
    {canManage&&editing!==undefined&&<LaunchForm item={editing||undefined} onClose={()=>setEditing(undefined)}/>}
  </>
}
