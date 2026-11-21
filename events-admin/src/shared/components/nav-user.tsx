import { ChevronsUpDown, LogIn, LogOut, Settings, UserCircle, UserRound } from 'lucide-react'
import { useState } from 'react'
import { UserProfileModalControlled } from '@/features/users'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { useAuth } from '@/shared/lib/auth'
import { getUserDisplayName, getUserEmail, getUserInitials } from '@/shared/lib/auth-utils'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/shared/ui/components/sidebar'

export function NavUser() {
  const { isMobile } = useSidebar()
  const { session, offlineIdentity, logout, isGuest, exitGuestMode, login } = useAuth()
  const { user } = useCurrentUser()
  const [profileOpen, setProfileOpen] = useState(false)

  const displayName = offlineIdentity?.displayName ?? getUserDisplayName(session)
  const initials = offlineIdentity?.displayName.slice(0, 2).toUpperCase() ?? getUserInitials(session)
  const email = offlineIdentity?.email ?? getUserEmail(session)

  const handleLogout = async () => {
    await logout()
  }

  const handleSignIn = async () => {
    exitGuestMode()
    await login()
  }

  if (isGuest) {
    return (
      <SidebarMenu className='border-t border-sidebar-border pt-2'>
        <SidebarMenuItem>
          <SidebarMenuButton
            size='lg'
            tooltip='Guest - Sign In'
            onClick={handleSignIn}
            aria-label='Sign in to your account'
          >
            <Avatar className='size-10 shrink-0 rounded-lg'>
              <AvatarFallback className='rounded-lg bg-muted'>
                <UserRound className='size-5' />
              </AvatarFallback>
            </Avatar>
            <div className='grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden'>
              <span className='truncate font-semibold'>Guest</span>
              <span className='truncate text-xs text-muted-foreground flex items-center gap-1'>
                <LogIn className='h-3 w-3' />
                Sign In
              </span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    )
  }

  return (
    <SidebarMenu className='border-t border-sidebar-border pt-2'>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size='lg'
                tooltip={displayName}
                aria-label={`User menu for ${displayName}`}
                className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
              />
            }
          >
            <Avatar className='size-10 shrink-0 rounded-lg'>
              <AvatarImage src='' alt={displayName} />
              <AvatarFallback className='rounded-lg bg-primary/10 text-primary'>{initials}</AvatarFallback>
            </Avatar>
            <div className='grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden'>
              <span className='flex items-center gap-1 min-w-0 font-semibold'>
                <span className='truncate'>{displayName}</span>
                {user && <AchievementNameBadge badge={user.selectedBadge} userId={user.id} inline />}
              </span>
              <span className='truncate text-xs text-muted-foreground'>{email}</span>
            </div>
            <ChevronsUpDown className='ml-auto size-4 group-data-[collapsible=icon]:hidden' />
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className='w-(--anchor-width) min-w-56 rounded-lg'
            side={isMobile ? 'bottom' : 'right'}
            align='end'
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className='p-0 font-normal'>
                <div className='flex items-center gap-2 px-1 py-1.5 text-left text-sm'>
                  <Avatar className='size-10 rounded-lg'>
                    <AvatarImage src='' alt={displayName} />
                    <AvatarFallback className='rounded-lg bg-primary/10 text-primary'>{initials}</AvatarFallback>
                  </Avatar>
                  <div className='grid flex-1 text-left text-sm leading-tight'>
                    <span className='flex items-center gap-1 min-w-0 font-semibold'>
                      <span className='truncate'>{displayName}</span>
                      {user && <AchievementNameBadge badge={user.selectedBadge} userId={user.id} />}
                    </span>
                    <span className='truncate text-xs text-muted-foreground'>{email}</span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={() => setProfileOpen(true)}>
                <UserCircle />
                Profile
              </DropdownMenuItem>
              <DropdownMenuItem>
                <Settings />
                Settings
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout} className='text-destructive focus:text-destructive'>
              <LogOut />
              Log out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <UserProfileModalControlled
          userId={profileOpen ? (user?.id ?? null) : null}
          onClose={() => setProfileOpen(false)}
        />
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
