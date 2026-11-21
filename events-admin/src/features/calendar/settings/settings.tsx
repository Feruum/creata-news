import { SettingsIcon } from 'lucide-react'
import { useTheme } from 'next-themes'
import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import { useDragDrop } from '@/features/calendar/contexts/use-drag-drop'
import { Button } from '@/shared/ui/components/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger
} from '@/shared/ui/components/dropdown-menu'
import { Switch } from '@/shared/ui/components/switch'

export function Settings() {
  const { badgeVariant, setBadgeVariant, use24HourFormat, toggleTimeFormat, agendaModeGroupBy, setAgendaModeGroupBy } =
    useCalendar()
  const { showConfirmation, setShowConfirmation } = useDragDrop()
  const { theme, setTheme } = useTheme()

  const isDarkMode = theme === 'dark'
  const isDotVariant = badgeVariant === 'dot'

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant='outline' size='icon' />}>
        <SettingsIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent className='w-56'>
        <DropdownMenuGroup>
          <DropdownMenuLabel>Calendar settings</DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem>
            Use dark mode
            <DropdownMenuShortcut>
              <Switch checked={isDarkMode} onCheckedChange={(checked) => setTheme(checked ? 'dark' : 'light')} />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem>
            Show confirmation dialog on event drop
            <DropdownMenuShortcut>
              <Switch checked={showConfirmation} onCheckedChange={(checked) => setShowConfirmation(checked)} />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem>
            Use dot badge
            <DropdownMenuShortcut>
              <Switch
                checked={isDotVariant}
                onCheckedChange={(checked) => setBadgeVariant(checked ? 'dot' : 'colored')}
              />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
          <DropdownMenuItem>
            Use 24 hour format
            <DropdownMenuShortcut>
              <Switch checked={use24HourFormat} onCheckedChange={toggleTimeFormat} />
            </DropdownMenuShortcut>
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel>Agenda view group by</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={agendaModeGroupBy}
            onValueChange={(value) => setAgendaModeGroupBy(value as 'date' | 'category')}
          >
            <DropdownMenuRadioItem value='date'>Date</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value='category'>Category</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
