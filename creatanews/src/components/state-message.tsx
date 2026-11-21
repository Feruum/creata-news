'use client'
import { ArrowUpRight, RefreshCw } from 'lucide-react'
import Link from 'next/link'
import { copy, localePath } from '@/lib/i18n'
import type { Locale } from '@/lib/models'
export function StateMessage({locale,error=false,title,text,home=false,compact=false}: {locale:Locale;error?:boolean;title?:string;text?:string;home?:boolean;compact?:boolean}) {const t=copy[locale];return <div className={`state-message${compact?' state-message-compact':''}`} role={error?'alert':'status'}><span className='state-mark'>{error?'↻':'✳'}</span><h2>{title||(error?t.errorTitle:t.emptyTitle)}</h2><p>{text||(error?t.errorText:t.emptyText)}</p>{error&&<button className='button button-outline' onClick={()=>window.location.reload()}><RefreshCw size={16}/>{t.retry}</button>}{home&&<Link href={localePath(locale)} className='button button-outline'>{t.backHome}<ArrowUpRight size={16}/></Link>}</div>}
