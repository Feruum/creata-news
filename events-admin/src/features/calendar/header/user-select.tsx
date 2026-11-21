import { useCalendar } from '@/features/calendar/contexts/use-calendar'
import { Avatar, AvatarFallback, AvatarImage } from '@/shared/ui/components/avatar'
import { AvatarGroup } from '@/shared/ui/components/avatar-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'

export function UserSelect() {
  const { users, selectedUserId, filterEventsBySelectedUser } = useCalendar()

  return (
    <Select
      value={selectedUserId!}
      onValueChange={(value) => {
        if (value !== null) filterEventsBySelectedUser(value)
      }}
    >
      <SelectTrigger className='w-full'>
        <SelectValue placeholder='Select a user' />
      </SelectTrigger>
      <SelectContent align='end'>
        <SelectItem value='all'>
          <AvatarGroup className='mx-2 flex items-center' max={3}>
            {users.map((user) => (
              <Avatar key={user.id} className='size-6 text-xxs'>
                <AvatarImage src={user.picturePath ?? undefined} alt={user.name} />
                <AvatarFallback className='text-xxs'>{user.name[0]}</AvatarFallback>
              </Avatar>
            ))}
          </AvatarGroup>
          All
        </SelectItem>

        {users.map((user) => (
          <SelectItem key={user.id} value={user.id} className='flex-1 cursor-pointer'>
            <div className='flex items-center gap-2'>
              <Avatar key={user.id} className='size-6'>
                <AvatarImage src={user.picturePath ?? undefined} alt={user.name} />
                <AvatarFallback className='text-xxs'>{user.name[0]}</AvatarFallback>
              </Avatar>

              <p className='truncate'>{user.name}</p>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
