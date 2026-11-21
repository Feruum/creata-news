import { Monitor, Moon, Sun } from 'lucide-react'
import { trackProductEvent } from '@/shared/lib/telemetry'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/shared/ui/components/sidebar'
import type { Theme } from '@/shared/ui/theme-provider-context'
import { useTheme } from '@/shared/ui/hooks/use-theme'

const labels: Record<Theme, string> = { light: 'Light', dark: 'Dark', system: 'System' }

export function SidebarThemeMenu() {
  const { theme, setTheme } = useTheme()
  const { isMobile } = useSidebar()
  const Icon = theme === 'system' ? Monitor : theme === 'dark' ? Moon : Sun
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger render={<SidebarMenuButton tooltip='Theme' aria-label={`Theme: ${labels[theme]}`} />}>
            <Icon />
            <span className='group-data-[collapsible=icon]:hidden'>Theme</span>
            <span className='ml-auto text-xs text-muted-foreground group-data-[collapsible=icon]:hidden'>
              {labels[theme]}
            </span>
          </DropdownMenuTrigger>
          <DropdownMenuContent side={isMobile ? 'top' : 'right'} align='end'>
            <DropdownMenuRadioGroup
              value={theme}
              onValueChange={(value) => {
                if (value !== 'light' && value !== 'dark' && value !== 'system') return
                setTheme(value)
                trackProductEvent('theme_change', { surface: 'sidebar', source: value, outcome: 'success' })
              }}
            >
              <DropdownMenuRadioItem value='light'>Light</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value='dark'>Dark</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value='system'>System</DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
