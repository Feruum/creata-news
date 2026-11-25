import type { ComponentProps } from 'react'
import { Activity, ArrowUpRight, Bot, CalendarClock, Clock3, FileText, Gauge, Layers3, ListTodo, LogOut, Radio, Settings2, ShieldCheck, Sparkles, Users } from 'lucide-react'
import { CreataBrand,CreataMark } from '@creata/brand/logo'
import { NavMain } from '@/shared/components/nav-main'
import { NavDocuments } from '@/shared/components/nav-documents'
import { NavSecondary } from '@/shared/components/nav-secondary'
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/shared/ui/components/sidebar'
import { useAdmin } from '../../creata/hooks'
import { canAccessStoryFollowUps, canViewAudit } from '../../creata/domain'
import type { CopyKey } from '../../creata/i18n'
import {adminPath,applicationUrls} from '../../creata/demo'

const editorial=[{url:'/today',key:'today',icon:Gauge},{url:'/bank',key:'bank',icon:Layers3},{url:'/articles',key:'articles',icon:FileText},{url:'/story-follow-ups',key:'storyFollowUps',icon:CalendarClock},{url:'/charts',key:'charts',icon:Activity},{url:'/launches',key:'launches',icon:ArrowUpRight},{url:'/newspapers',key:'newspapers',icon:FileText},{url:'/opportunities',key:'opportunities',icon:Sparkles},{url:'/digests',key:'digests',icon:Radio},{url:'/authors',key:'authors',icon:Users},{url:'/analytics',key:'analytics',icon:Activity},{url:'/content',key:'content',icon:FileText},{url:'/media',key:'mediaLibrary',icon:FileText},{url:'/settings',key:'settings',icon:Settings2},{url:'/audit',key:'audit',icon:Clock3}] as const
const hermes=[{url:'/sources',key:'sources',icon:Radio},{url:'/runs',key:'runs',icon:Activity},{url:'/jobs',key:'jobs',icon:ListTodo},{url:'/charts',key:'charts',icon:Activity},{url:'/rules',key:'rules',icon:ShieldCheck},{url:'/ai',key:'ai',icon:Sparkles},{url:'/errors',key:'errors',icon:Activity},{url:'/settings',key:'settings',icon:Settings2}] as const
export function AppSidebar({onLogout,...props}:ComponentProps<typeof Sidebar>&{onLogout:()=>void}) {
 const { t, locale, mode, session } = useAdmin()
 const user = session?.user
 const isAdmin = user?.roles.includes('admin')
 const isHermes = mode === 'hermes'
 const items = (isHermes ? hermes : editorial)
  .filter(item => item.key !== 'storyFollowUps' || canAccessStoryFollowUps(user))
  .filter(item => item.key !== 'audit' || canViewAudit(user))
  .map(item => ({ url: item.url, title: t(item.key), icon: item.icon }))
 const relatedApps = [
  ...(isAdmin ? [{
   title: t(isHermes ? 'openEditorial' : 'openHermes'),
   url: isHermes ? applicationUrls.editorial : applicationUrls.hermes,
   external: true,
   icon: Bot,
  }] : []),
  { title: t('openSite'), url: applicationUrls.web + '/' + locale, external: true, icon: ArrowUpRight },
 ]

 return <Sidebar collapsible='icon' {...props}>
  <SidebarHeader>
   <SidebarMenu>
    <SidebarMenuItem>
     <SidebarMenuButton asChild className='hover:bg-transparent! h-11' tooltip='Creata News'>
      <a href={adminPath('/')}>
       <CreataBrand className='admin-creata-logo'/>
       <CreataMark className='admin-creata-mark'/>
      </a>
     </SidebarMenuButton>
    </SidebarMenuItem>
   </SidebarMenu>
  </SidebarHeader>
  <SidebarContent>
   <NavMain items={items.slice(0, 3)} label={t(isHermes ? 'hermes' : 'newsroom')}
    createLabel={t(isHermes ? 'newSource' : 'newArticle')}
    createUrl={isHermes ? '/sources?new=1' : '/articles?new=1'}/>
   <NavDocuments label={t('management')} items={items.slice(3).map(item => ({ name: item.title, url: item.url, icon: item.icon }))}/>
   <div className='mt-auto'><NavSecondary items={relatedApps}/></div>
  </SidebarContent>
  <SidebarFooter>
   <SidebarMenu>
    <SidebarMenuItem>
     <SidebarMenuButton tooltip={t('logout')} aria-label={t('logout')} className='h-12' onClick={onLogout}>
      <span className='admin-user-avatar'>{user?.name.charAt(0) || 'C'}</span>
      <span className='admin-user-copy'>
       <strong>{user?.name}</strong>
       <small>{user?.roles.map(role => t(role as CopyKey)).join(', ')}</small>
      </span>
      <LogOut className='ml-auto size-4'/>
     </SidebarMenuButton>
    </SidebarMenuItem>
   </SidebarMenu>
  </SidebarFooter>
 </Sidebar>
}
