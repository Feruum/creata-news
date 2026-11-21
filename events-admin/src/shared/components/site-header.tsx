import { useRouterState } from '@tanstack/react-router'
import { Moon, Sun, Search } from 'lucide-react'
import { Breadcrumb, BreadcrumbItem, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from '@/shared/ui/components/breadcrumb'
import { Separator } from '@/shared/ui/components/separator'
import { SidebarTrigger } from '@/shared/ui/components/sidebar'
import { Button } from '@/shared/ui/components/button'
import { Input } from '@/shared/ui/components/input'
import { useAdmin } from '../../creata/hooks'
import type { Locale } from '../../creata/types'
import { hasCopy } from '../../creata/i18n'
import {adminPath,adminRoutePath} from '../../creata/demo'
export function SiteHeader({dark,onToggleTheme}:{dark:boolean;onToggleTheme:()=>void}) {
 const {t,mode,locale,setLocale}=useAdmin();const path=useRouterState({select:s=>s.location.pathname});const routeKey=adminRoutePath(path).split('/')[1]||'today';const key=routeKey==='media'?'mediaLibrary':routeKey
 return <header className='original-site-header flex h-(--header-height) shrink-0 items-center border-b bg-background sticky top-0 z-30'><div className='flex w-full min-w-0 items-center gap-3 px-4 lg:px-6'><SidebarTrigger aria-label={t('menu')} className='size-9 shrink-0'/><Separator orientation='vertical' className='h-4 mx-1 hidden sm:block'/><Breadcrumb className='hidden sm:block'><BreadcrumbList><BreadcrumbItem>{t(mode==='hermes'?'hermes':'newsroom')}</BreadcrumbItem><BreadcrumbSeparator/><BreadcrumbItem><BreadcrumbPage>{hasCopy(key)?t(key):t('articles')}</BreadcrumbPage></BreadcrumbItem></BreadcrumbList></Breadcrumb><div className='ml-auto flex items-center gap-3'>{mode==='editorial'&&<form action={adminPath('/articles')} className='original-header-search hidden lg:flex items-center gap-2'><Search className='size-4 text-muted-foreground'/><Input name='q' aria-label={t('search')} placeholder={`${t('search')}…`} className='h-8 w-52 border-0 shadow-none'/></form>}<select className='original-locale-select' aria-label={t('language')} value={locale} onChange={e=>setLocale(e.target.value as Locale)}><option value='ru'>RU</option><option value='kk'>ҚАЗ</option><option value='en'>EN</option></select><Button variant='ghost' size='icon-sm' aria-label={t('toggleTheme')} onClick={onToggleTheme}>{dark?<Sun/>:<Moon/>}</Button></div></div></header>
}
