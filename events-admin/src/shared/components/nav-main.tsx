import { Link, useMatchRoute } from '@tanstack/react-router'
import { CirclePlus, type LucideIcon } from 'lucide-react'
import {adminPath} from '../../creata/demo'
import { SidebarGroup, SidebarGroupContent, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/shared/ui/components/sidebar'

export function NavMain({ items, label, createLabel, createUrl }: { items: {title:string;url:string;icon?:LucideIcon}[]; label:string; createLabel?:string; createUrl?:string }) {
 const matchRoute=useMatchRoute();const {setOpenMobile}=useSidebar()
 return <SidebarGroup><SidebarGroupContent className='flex flex-col gap-2'>{createUrl&&<SidebarMenu><SidebarMenuItem><SidebarMenuButton asChild tooltip={createLabel} variant='subtle'><a href={adminPath(createUrl)}><CirclePlus/><span>{createLabel}</span></a></SidebarMenuButton></SidebarMenuItem></SidebarMenu>}<SidebarGroupLabel>{label}</SidebarGroupLabel><SidebarMenu>{items.map(item=><SidebarMenuItem key={item.url}><SidebarMenuButton asChild tooltip={item.title} isActive={!!matchRoute({to:item.url,fuzzy:true})}><Link to={item.url} onClick={()=>setOpenMobile(false)}>{item.icon&&<item.icon/>}<span>{item.title}</span></Link></SidebarMenuButton></SidebarMenuItem>)}</SidebarMenu></SidebarGroupContent></SidebarGroup>
}
