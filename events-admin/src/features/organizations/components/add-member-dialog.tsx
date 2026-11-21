import type { RoleTemplate, User } from '@studyverse/proto'
import { Loader2, Search, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { useAddOrganizationMember } from '@/features/organizations'
import { AchievementNameBadge } from '@/shared/components/achievement-name-badge'
import { Avatar, AvatarFallback } from '@/shared/ui/components/avatar'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { useMemberLookup } from '../api/queries'
import { membershipExpiryValue } from '../lib/membership-expiry'

interface AddMemberDialogProps {
  organizationId: number
  roleTemplates: RoleTemplate[]
  existingMemberUserIds: number[]
}

export function AddMemberDialog({ organizationId, roleTemplates, existingMemberUserIds }: AddMemberDialogProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [lookup, setLookup] = useState('')
  const [validUntil, setValidUntil] = useState('')
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [selectedRoleId, setSelectedRoleId] = useState<string>('')
  const { data: usersData, isFetching: isLoadingUsers, error: lookupError } = useMemberLookup(lookup, open)
  const users = usersData?.user ? [usersData.user] : []

  const addMemberMutation = useAddOrganizationMember(organizationId)

  const resetForm = () => {
    setSearch('')
    setLookup('')
    setValidUntil('')
    setSelectedUserId(null)
    setSelectedRoleId('')
  }

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen)
    if (!newOpen) {
      resetForm()
    }
  }

  const handleAddMember = () => {
    if (!selectedUserId || !selectedRoleId) {
      toast.error('Please select a user and a role')
      return
    }
    let expiry: string
    try {
      expiry = membershipExpiryValue(validUntil)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Invalid membership expiry')
      return
    }
    addMemberMutation.mutate(
      {
        userId: selectedUserId,
        roleTemplateId: parseInt(selectedRoleId, 10),
        validUntil: expiry || undefined
      },
      {
        onSuccess: () => {
          toast.success('Member added successfully')
          setOpen(false)
          resetForm()
        },
        onError: (error) => {
          toast.error(`Failed to add member: ${error.message}`)
        }
      }
    )
  }

  const filteredUsers =
    users.filter((user: User) => {
      if (existingMemberUserIds.includes(user.id)) return false
      return true
    }) || []

  const selectedUser = users.find((user: User) => user.id === selectedUserId)

  const sortedRoles = [...roleTemplates].sort((a, b) => a.sortOrder - b.sortOrder)

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger render={<Button variant='outline' size='sm' className='gap-2' />}>
        <UserPlus className='h-4 w-4' />
        Add Member
      </DialogTrigger>
      <DialogContent className='max-w-md'>
        <DialogHeader>
          <DialogTitle>Add Organization Member</DialogTitle>
          <DialogDescription>Find a user by their exact username or #ID and assign a role.</DialogDescription>
        </DialogHeader>

        <div className='space-y-4 py-4'>
          <div className='space-y-2'>
            <label htmlFor='organization-user-search' className='text-sm font-medium'>
              Search User
            </label>
            <div className='relative'>
              <Search className='absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground' />
              <Input
                id='organization-user-search'
                placeholder='@username or #user ID'
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value)
                  setLookup('')
                  setSelectedUserId(null)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    setLookup(search)
                  }
                }}
                className='pl-9'
              />
            </div>
            <Button
              type='button'
              variant='outline'
              onClick={() => setLookup(search)}
              disabled={!search.trim() || isLoadingUsers}
            >
              Find user
            </Button>
          </div>

          {isLoadingUsers ? (
            <div className='flex items-center justify-center py-8'>
              <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
            </div>
          ) : (
            <div className='max-h-48 space-y-1 overflow-y-auto rounded-md border p-2'>
              {filteredUsers.length === 0 ? (
                <p className='py-4 text-center text-sm text-muted-foreground'>
                  {lookupError
                    ? 'User lookup failed. Check the username or ID and try again.'
                    : lookup
                      ? users.length
                        ? 'This user is already a member'
                        : 'No user found'
                      : 'Enter a username or #ID, then choose Find user.'}
                </p>
              ) : (
                filteredUsers.map((user: User) => (
                  <UserOption
                    key={user.id}
                    user={user}
                    isSelected={selectedUserId === user.id}
                    onSelect={() => setSelectedUserId(user.id)}
                  />
                ))
              )}
            </div>
          )}

          {selectedUser && (
            <div className='rounded-md border bg-muted/50 p-3'>
              <p className='text-xs text-muted-foreground mb-1'>Selected user:</p>
              <div className='flex items-center gap-2'>
                <Avatar className='h-6 w-6'>
                  <AvatarFallback className='text-xs'>{getInitials(selectedUser)}</AvatarFallback>
                </Avatar>
                <span className='text-sm font-medium'>{getDisplayName(selectedUser)}</span>
              </div>
            </div>
          )}

          <div className='space-y-2'>
            <label htmlFor='organization-member-expiry' className='text-sm font-medium'>
              Member until (optional)
            </label>
            <Input
              id='organization-member-expiry'
              type='datetime-local'
              value={validUntil}
              onChange={(e) => setValidUntil(e.target.value)}
            />
            <p className='text-xs text-muted-foreground'>Leave empty for no expiry.</p>
          </div>
          <div className='space-y-2'>
            <label htmlFor='organization-role-select' className='text-sm font-medium'>
              Role
            </label>
            <Select
              value={selectedRoleId}
              onValueChange={(value) => {
                if (value !== null) setSelectedRoleId(value)
              }}
              disabled={sortedRoles.length === 0}
            >
              <SelectTrigger id='organization-role-select'>
                <SelectValue placeholder='Select a role...' />
              </SelectTrigger>
              <SelectContent>
                {sortedRoles.map((role) => (
                  <SelectItem key={role.id} value={role.id.toString()}>
                    {role.displayName}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {sortedRoles.length === 0 && (
              <p role='status' className='text-sm text-muted-foreground'>
                No roles are configured. Close this dialog and use Edit Roles to create an organization role, or
                configure Default Roles in Organization Types.
              </p>
            )}
          </div>
        </div>

        <div className='flex justify-end gap-2'>
          <Button variant='outline' onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleAddMember}
            disabled={!selectedUserId || !selectedRoleId || addMemberMutation.isPending}
          >
            {addMemberMutation.isPending && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            Add Member
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

interface UserOptionProps {
  user: User
  isSelected: boolean
  onSelect: () => void
}

function UserOption({ user, isSelected, onSelect }: UserOptionProps) {
  const displayName = getDisplayName(user)
  const initials = getInitials(user)

  return (
    <div
      className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left transition-colors ${
        isSelected ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
      }`}
    >
      <button type='button' onClick={onSelect} className='flex min-w-0 flex-1 items-center gap-3 text-left'>
        <Avatar className='h-8 w-8'>
          <AvatarFallback className={`text-xs ${isSelected ? 'bg-primary-foreground/20' : ''}`}>
            {initials}
          </AvatarFallback>
        </Avatar>
        <div className='min-w-0 flex-1'>
          <p className='truncate text-sm font-medium'>{displayName}</p>
          <p className={`truncate text-xs ${isSelected ? 'text-primary-foreground/70' : 'text-muted-foreground'}`}>
            {user.email}
          </p>
        </div>
      </button>
      <AchievementNameBadge badge={user.selectedBadge} userId={user.id} />
    </div>
  )
}

function getDisplayName(user: User): string {
  if (user.firstName && user.lastName) {
    return `${user.firstName} ${user.lastName}`
  }
  return user.username || user.email
}

function getInitials(user: User): string {
  const name = getDisplayName(user)
  return name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}
