import { useLayoutEffect, useState } from 'react'
import { Outlet, useRouterState } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CreataBrand } from '@creata/brand/logo'
import { ShieldCheck } from 'lucide-react'
import { AppSidebar } from '@/shared/components/app-sidebar'
import { SiteHeader } from '@/shared/components/site-header'
import { SidebarProvider, SidebarInset } from '@/shared/ui/components/sidebar'
import { useAdmin } from './hooks'
import { Button, ErrorPanel, Field, Loading } from './ui'
import type { CopyKey } from './i18n'
import {adminDemo,applicationUrls} from './demo'
import {DemoBar} from './demo-bar'
import { normalizeAdminTheme } from './admin-theme'
function Login(){
 const {t,session,api,refreshSession}=useAdmin();const [role,setRole]=useState('chief_editor');const returnTo=encodeURIComponent(window.location.origin+window.location.pathname)
 const login=useMutation({mutationFn:()=>api('/auth/dev-login',{method:'POST',body:JSON.stringify({role})}),onSuccess:refreshSession})
 return <main className='login-screen'><section className='login-form'><a className='login-creata-brand' href={applicationUrls.web||'/'}><CreataBrand/></a><h1>{t('loginLead')}</h1><div className='login-buttons'>{session?.providers.google&&<a className='button secondary' href={`/api/creata/v1/auth/google/start?returnTo=${returnTo}`}>{t('googleLogin')}</a>}{session?.providers.telegram&&<a className='button secondary' href={`/api/creata/v1/auth/telegram/start?returnTo=${returnTo}`}>{t('telegramLogin')}</a>}{!session?.providers.google&&!session?.providers.telegram&&<p className='callout'>{t('loginUnconfigured')}</p>}</div>{session?.devAuth&&<div className='dev-login'><small>{t('devLabel')}</small><Field label={t('role')}><select value={role} onChange={e=>setRole(e.target.value)}>{['reader','author','editor','chief_editor','admin'].map(r=><option key={r} value={r}>{t(r as CopyKey)}</option>)}</select></Field><Button disabled={login.isPending} onClick={()=>login.mutate()}>{t('devLogin')}</Button>{login.error&&<ErrorPanel error={login.error}/>}</div>}</section></main>
}
export function Shell(){
 const {t,mode,session,sessionLoading,sessionError,refreshSession,api}=useAdmin();const client=useQueryClient();const path=useRouterState({select:s=>s.location.pathname});const theme=normalizeAdminTheme(session?.user?.adminTheme)
 const logout=useMutation({mutationFn:()=>api('/auth/logout',{method:'POST'}),onSuccess:()=>{client.setQueryData(['session'],{...session,user:null,csrfToken:undefined});client.removeQueries({predicate:q=>q.queryKey[0]!=='session'});refreshSession()}})
 useLayoutEffect(()=>{document.documentElement.dataset.theme=theme;document.documentElement.classList.toggle('dark',theme==='dark')},[theme])
 if(sessionLoading)return <Loading/>;if(sessionError)return <div className='login-error'><ErrorPanel error={sessionError} retry={refreshSession}/></div>;if(!session?.user)return <Login/>
 const user=session.user;if(!user.roles.some(r=>['admin','chief_editor','editor','author'].includes(r))||(mode==='hermes'&&!user.roles.includes('admin')))return <main className='denied'><ShieldCheck size={36}/><h1>{t('deniedTitle')}</h1><p>{t('deniedLead')}</p><Button variant='secondary' onClick={()=>logout.mutate()}>{t('logout')}</Button></main>
 return <SidebarProvider className='h-svh overflow-hidden creata-admin-shell' style={{'--sidebar-width':'17rem','--header-height':'3.5rem'} as React.CSSProperties}><a className='skip-link' href='#main-content'>{t('open')}</a><AppSidebar variant='inset' onLogout={()=>logout.mutate()}/><SidebarInset><div className='admin-scroll'><SiteHeader/><div className='main-content' id='main-content' key={path}>{adminDemo&&<DemoBar/>}<Outlet/></div></div></SidebarInset></SidebarProvider>
}
