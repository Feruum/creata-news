import { PlatformRole, type User } from '@studyverse/proto'
import { ShieldIcon } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { useAssignPlatformRole } from '@/features/users'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import { Label } from '@/shared/ui/components/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'

interface AssignRoleDialogProps {
  user: User
  children?: React.ReactElement
}

function platformRoleToString(role: PlatformRole): string {
  switch (role) {
    case PlatformRole.ADMIN:
      return 'admin'
    case PlatformRole.STAFF:
      return 'staff'
    default:
      return 'user'
  }
}

function stringToPlatformRole(role: string): PlatformRole {
  switch (role) {
    case 'admin':
      return PlatformRole.ADMIN
    case 'staff':
      return PlatformRole.STAFF
    default:
      return PlatformRole.USER
  }
}

export function AssignRoleDialog({ user, children }: AssignRoleDialogProps) {
  const [open, setOpen] = useState(false)
  const [role, setRole] = useState<string>(platformRoleToString(user.platformRole))

  const mutation = useAssignPlatformRole()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await mutation.mutateAsync({
        userId: user.id,
        role: stringToPlatformRole(role)
      })
      toast.success(`Role updated for ${user.username}`)
      setOpen(false)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to assign role')
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        onClick={(event) => event.stopPropagation()}
        render={
          children || (
            <Button variant='ghost' size='sm'>
              <ShieldIcon className='mr-1.5 h-3.5 w-3.5' />
              Assign Role
            </Button>
          )
        }
      ></DialogTrigger>
      <DialogContent className='sm:max-w-md' onClick={(event) => event.stopPropagation()}>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Assign Role</DialogTitle>
            <DialogDescription>
              Change platform role for <strong>{user.username}</strong> ({user.email})
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='role'>Platform Role</Label>
              <Select
                value={role}
                onValueChange={(value) => {
                  if (value !== null) setRole(value)
                }}
              >
                <SelectTrigger id='role'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='user'>User</SelectItem>
                  <SelectItem value='staff'>Staff</SelectItem>
                  <SelectItem value='admin'>Administrator</SelectItem>
                </SelectContent>
              </Select>
              <p className='text-xs text-muted-foreground'>
                Users have no special permissions. Staff can manage organizations and view analytics. Administrators
                have full system access.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type='submit' disabled={mutation.isPending}>
              {mutation.isPending ? 'Saving...' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
