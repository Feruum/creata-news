'use client'

import { SearchResultType } from '@studyverse/proto'
import { useNavigate } from '@tanstack/react-router'
import {
  Award,
  Building2,
  Calendar,
  CalendarCheck,
  ChartNoAxesCombined,
  FileStack,
  LayoutDashboard,
  Loader2,
  Moon,
  Plus,
  Search,
  Settings,
  Sun,
  Upload,
  User,
  Users
} from 'lucide-react'
import * as React from 'react'
import { useSearchResults } from '@/features/search'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { useDebounce } from '@/shared/hooks/use-debounce'
import { trackProductEvent } from '@/shared/lib/telemetry'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut
} from '@/shared/ui/components/command'
import { useTheme } from '@/shared/ui/hooks/use-theme'
import { matchesCommand } from '../lib/command-matching'

const navigationItems = [
  { name: 'Dashboard', icon: LayoutDashboard, path: '/', shortcut: 'G D' },
  { name: 'Calendar', icon: CalendarCheck, path: '/calendar', shortcut: 'G C' },
  { name: 'Events', icon: Calendar, path: '/events', shortcut: 'G E' },
  { name: 'Organizations', icon: Building2, path: '/organizations', shortcut: 'G O' },
  { name: 'Users', icon: Users, path: '/users', shortcut: 'G U' },
  { name: 'Reports', icon: FileStack, path: '/reports', shortcut: 'G R' }
]

const resultTypeConfig: Partial<Record<SearchResultType, { icon: typeof Calendar; label: string; path: string }>> = {
  [SearchResultType.EVENT]: { icon: Calendar, label: 'Event', path: '/events' },
  [SearchResultType.ORGANIZATION]: { icon: Building2, label: 'Organization', path: '/organizations' },
  [SearchResultType.USER]: { icon: User, label: 'User', path: '/users' },
  [SearchResultType.UNSPECIFIED]: { icon: Search, label: 'Unknown', path: '/' }
}

export default function SearchCommandPalette() {
  const [open, setOpen] = React.useState(false)
  const [searchQuery, setSearchQuery] = React.useState('')
  const debouncedQuery = useDebounce(searchQuery, 300)
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const { isAdmin, isStaff } = useCurrentUser()

  const { data: searchResults, isLoading: isSearching } = useSearchResults(debouncedQuery)

  React.useEffect(() => {
    if (open && debouncedQuery.length >= 2) trackProductEvent('search', { surface: 'search' })
  }, [open, debouncedQuery])

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  React.useEffect(() => {
    if (!open) {
      setSearchQuery('')
    }
  }, [open])

  const runCommand = React.useCallback((command: () => void) => {
    setOpen(false)
    command()
  }, [])

  const handleSearchResultSelect = React.useCallback(
    (type: SearchResultType, id: number) => {
      const config = resultTypeConfig[type] ?? resultTypeConfig[SearchResultType.UNSPECIFIED]
      if (!config) return
      runCommand(() => {
        navigate({ to: `${config.path}/${id}` })
      })
    },
    [navigate, runCommand]
  )

  const hasSearchResults = searchResults?.results && searchResults.results.length > 0
  const isSearchMode = searchQuery.length >= 2
  const showSearchResults = isSearchMode && debouncedQuery.length >= 2
  const coreNavigation = navigationItems.flatMap((item) =>
    item.path === '/' && (isAdmin || isStaff)
      ? [item, { name: 'Analytics', icon: ChartNoAxesCombined, path: '/analytics', shortcut: '' }]
      : [item]
  )
  const permittedNavigation = isAdmin
    ? [
        ...coreNavigation,
        { name: 'Achievements', icon: Award, path: '/achievements', shortcut: 'G A' },
        { name: 'Schedules', icon: CalendarCheck, path: '/schedules', shortcut: 'G S' }
      ]
    : coreNavigation
  const matchingNavigation = permittedNavigation.filter((item) =>
    matchesCommand(searchQuery, item.name, item.path, item.shortcut)
  )
  const showCreate = matchesCommand(searchQuery, 'Create Event', 'C E')
  const showImport = matchesCommand(searchQuery, 'Import from Excel', 'C I')
  const showLight = matchesCommand(searchQuery, 'Light Mode')
  const showDark = matchesCommand(searchQuery, 'Dark Mode')
  const showSystem = matchesCommand(searchQuery, 'System Theme')
  const hasLocalResults =
    matchingNavigation.length > 0 || showCreate || showImport || showLight || showDark || showSystem

  return (
    <>
      <button
        type='button'
        onClick={() => setOpen(true)}
        className='inline-flex h-10 w-full min-w-0 max-w-lg items-center justify-between rounded-lg border border-input bg-background px-4 text-sm text-muted-foreground transition-colors hover:bg-accent/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2'
        aria-label='Open search (⌘K)'
      >
        <span className='truncate'>Search...</span>
        <span className='ml-4 flex-shrink-0 text-xs text-muted-foreground/60'>⌘K</span>
      </button>

      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        shouldFilter={false}
        title='Search'
        description='Search for commands, pages, or content'
        className='max-w-2xl'
      >
        <CommandInput placeholder='Type a command or search...' value={searchQuery} onValueChange={setSearchQuery} />
        <CommandList className='max-h-[400px]'>
          {isSearching && showSearchResults && (
            <div className='flex items-center justify-center py-6'>
              <Loader2 className='size-4 animate-spin text-muted-foreground' />
              <span className='ml-2 text-sm text-muted-foreground'>Searching...</span>
            </div>
          )}

          {showSearchResults && !isSearching && hasSearchResults && (
            <>
              <CommandGroup heading={`Search Results (${searchResults.totalHits})`} forceMount>
                {searchResults.results.map((result) => {
                  const config = resultTypeConfig[result.type] ?? resultTypeConfig[SearchResultType.UNSPECIFIED]
                  if (!config) return null
                  const Icon = config.icon
                  return (
                    <CommandItem
                      key={`${result.type}-${result.id}`}
                      value={`search-${result.type}-${result.id}-${result.title}`}
                      onSelect={() => handleSearchResultSelect(result.type, result.id)}
                      forceMount
                    >
                      <Icon aria-hidden='true' className='mr-2 size-4 opacity-60' />
                      <div className='flex flex-col min-w-0 flex-1'>
                        <span className='flex items-center gap-1.5'>
                          <span className='truncate'>{result.title}</span>
                          {result.type === SearchResultType.USER && (
                            <AchievementNameBadge badge={result.selectedBadge} userId={result.id} />
                          )}
                        </span>
                        {result.description && (
                          <span className='text-xs text-muted-foreground line-clamp-1'>{result.description}</span>
                        )}
                      </div>
                      <CommandShortcut>{config.label}</CommandShortcut>
                    </CommandItem>
                  )
                })}
              </CommandGroup>
              <CommandSeparator />
            </>
          )}

          {showSearchResults && !isSearching && !hasSearchResults && !hasLocalResults && (
            <CommandEmpty>No results found for "{debouncedQuery}"</CommandEmpty>
          )}

          {!isSearchMode && !hasLocalResults && <CommandEmpty>No results found.</CommandEmpty>}

          {matchingNavigation.length > 0 && (
            <CommandGroup heading='Navigation'>
              {matchingNavigation.map((item) => (
                <CommandItem key={item.path} onSelect={() => runCommand(() => navigate({ to: item.path }))}>
                  <item.icon aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>{item.name}</span>
                  <CommandShortcut>{item.shortcut}</CommandShortcut>
                </CommandItem>
              ))}
            </CommandGroup>
          )}

          {matchingNavigation.length > 0 && <CommandSeparator />}

          {(showCreate || showImport) && (
            <CommandGroup heading='Quick Actions'>
              {showCreate && (
                <CommandItem
                  onSelect={() =>
                    runCommand(() => {
                      navigate({ to: '/events', search: { action: 'create' } })
                    })
                  }
                >
                  <Plus aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>Create Event</span>
                  <CommandShortcut>C E</CommandShortcut>
                </CommandItem>
              )}
              {showImport && (
                <CommandItem
                  onSelect={() =>
                    runCommand(() => {
                      navigate({ to: '/events', search: { action: 'import' } })
                    })
                  }
                >
                  <Upload aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>Import from Excel</span>
                  <CommandShortcut>C I</CommandShortcut>
                </CommandItem>
              )}
            </CommandGroup>
          )}

          {(showCreate || showImport) && <CommandSeparator />}

          {(showLight || showDark || showSystem) && (
            <CommandGroup heading='Theme'>
              {showLight && (
                <CommandItem onSelect={() => runCommand(() => setTheme('light'))}>
                  <Sun aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>Light Mode</span>
                  {theme === 'light' && <CommandShortcut>✓</CommandShortcut>}
                </CommandItem>
              )}
              {showDark && (
                <CommandItem onSelect={() => runCommand(() => setTheme('dark'))}>
                  <Moon aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>Dark Mode</span>
                  {theme === 'dark' && <CommandShortcut>✓</CommandShortcut>}
                </CommandItem>
              )}
              {showSystem && (
                <CommandItem onSelect={() => runCommand(() => setTheme('system'))}>
                  <Settings aria-hidden='true' className='mr-2 size-4 opacity-60' />
                  <span>System Theme</span>
                  {theme === 'system' && <CommandShortcut>✓</CommandShortcut>}
                </CommandItem>
              )}
            </CommandGroup>
          )}
        </CommandList>
      </CommandDialog>
    </>
  )
}
