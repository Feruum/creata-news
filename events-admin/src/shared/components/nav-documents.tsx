import { Link, useMatchRoute } from '@tanstack/react-router'
import { ChevronRight, type LucideIcon } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/shared/ui/components/collapsible'
import { SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/shared/ui/components/sidebar'
export function NavDocuments({items,label}:{items:{name:string;url:string;icon:LucideIcon}[];label:string}) {
 const matchRoute=useMatchRoute();const {setOpenMobile}=useSidebar()
 return <Collapsible defaultOpen className='group/collapsible'><SidebarGroup><SidebarGroupLabel asChild><CollapsibleTrigger className='flex w-full items-center justify-between'>{label}<ChevronRight className='size-4 transition-transform group-data-open/collapsible:rotate-90'/></CollapsibleTrigger></SidebarGroupLabel><CollapsibleContent><SidebarMenu>{items.map(item=><SidebarMenuItem key={item.url}><SidebarMenuButton asChild tooltip={item.name} isActive={!!matchRoute({to:item.url,fuzzy:true})}><Link to={item.url} onClick={()=>setOpenMobile(false)}><item.icon/><span>{item.name}</span></Link></SidebarMenuButton></SidebarMenuItem>)}</SidebarMenu></CollapsibleContent></SidebarGroup></Collapsible>
}
