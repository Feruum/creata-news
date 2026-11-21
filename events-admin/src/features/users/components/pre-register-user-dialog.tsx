import { PlatformRole } from '@studyverse/proto'
import { useState } from 'react'
import { toast } from 'sonner'

import { usePreRegisterUser } from '@/features/users'
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
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'

interface PreRegisterUserDialogProps {
  children: React.ReactElement
}

export function PreRegisterUserDialog({ children }: PreRegisterUserDialogProps) {
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<'staff' | 'admin'>('staff')

  const mutation = usePreRegisterUser()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await mutation.mutateAsync({
        email,
        platformRole: role === 'admin' ? PlatformRole.ADMIN : PlatformRole.STAFF
      })
      toast.success('User pre-registered successfully')
      setOpen(false)
      setEmail('')
      setRole('staff')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to pre-register user')
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={children}></DialogTrigger>
      <DialogContent className='sm:max-w-md'>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Pre-register User</DialogTitle>
            <DialogDescription>
              Pre-register an email address so they automatically get assigned a role when they first sign up.
            </DialogDescription>
          </DialogHeader>
          <div className='grid gap-4 py-4'>
            <div className='grid gap-2'>
              <Label htmlFor='email'>Email</Label>
              <Input
                id='email'
                type='email'
                placeholder='user@example.com'
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className='grid gap-2'>
              <Label htmlFor='role'>Platform Role</Label>
              <Select
                value={role}
                onValueChange={(v) => {
                  if (v === null) return
                  return setRole(v as 'staff' | 'admin')
                }}
              >
                <SelectTrigger id='role'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value='staff'>Staff</SelectItem>
                  <SelectItem value='admin'>Administrator</SelectItem>
                </SelectContent>
              </Select>
              <p className='text-xs text-muted-foreground'>
                Staff can manage organizations and view analytics. Administrators have full system access.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button type='button' variant='outline' onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type='submit' disabled={mutation.isPending}>
              {mutation.isPending ? 'Pre-registering...' : 'Pre-register'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
