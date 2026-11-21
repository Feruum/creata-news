import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useAdmin, useCategories } from './hooks'
import { localized, validateSource } from './domain'
import { Button, ErrorPanel, Field, Modal } from './ui'
import type { Locale } from './types'
export function CollectLinkDialog({onClose}:{onClose:()=>void}){
 const {api,t,locale,toast}=useAdmin(),client=useQueryClient(),categories=useCategories();const [url,setUrl]=useState(''),[lang,setLang]=useState<Locale>(locale),[topics,setTopics]=useState<string[]>([]),[invalid,setInvalid]=useState(false)
 const mutation=useMutation({mutationFn:()=>api('/admin/bank/collect',{method:'POST',body:JSON.stringify({url,locale:lang,topics})}),onSuccess:()=>{void client.invalidateQueries({queryKey:['bank']});toast(t('sourceQueued'));onClose()}})
 return <Modal title={t('submitLink')} onClose={onClose}><form className='form-stack' onSubmit={e=>{e.preventDefault();const error=validateSource({name:'manual',type:'website',url,intervalMinutes:15});setInvalid(!!error);if(!error)mutation.mutate()}}><p className='panel-description'>{t('submitLinkHelp')}</p><Field label={t('url')}><input required type='url' value={url} onChange={e=>setUrl(e.target.value)} placeholder='https://example.com/story'/></Field><Field label={t('language')}><select value={lang} onChange={e=>setLang(e.target.value as Locale)}><option value='ru'>RU</option><option value='kk'>ҚАЗ</option><option value='en'>EN</option></select></Field><div className='field'><span>{t('topic')}</span><div className='topic-checks'>{categories.data?.items.map(c=><label className='topic-check' key={c.id}><input type='checkbox' checked={topics.includes(c.id)} onChange={()=>setTopics(v=>v.includes(c.id)?v.filter(x=>x!==c.id):[...v,c.id])}/>{localized(c.name,locale)}</label>)}</div></div>{invalid&&<p className='form-error'>{t('invalidUrl')}</p>}{mutation.error&&<ErrorPanel error={mutation.error}/>}<div className='form-actions'><Button type='button' variant='secondary' onClick={onClose}>{t('cancel')}</Button><Button disabled={mutation.isPending}>{t('submitLink')}</Button></div></form></Modal>
}
