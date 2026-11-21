import { Link, useMatchRoute } from '@tanstack/react-router'
import { Bot } from 'lucide-react'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem
} from '@/shared/ui/components/sidebar'
import { useHermesAccess } from '../hooks/use-hermes-access'

export function HermesNavigation() {
  const { state } = useHermesAccess()
  const matchRoute = useMatchRoute()
  if (state.status !== 'ready') return null

  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip='Hermes' isActive={!!matchRoute({ to: '/hermes', fuzzy: true })}>
              <Link to='/hermes'>
                <Bot />
                <span>Hermes</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
