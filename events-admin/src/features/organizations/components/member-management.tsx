import type { OrganizationMember, RoleTemplate } from '@studyverse/proto'
import { useState } from 'react'
import { toast } from 'sonner'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/shared/ui/components/alert-dialog'
import { Button } from '@/shared/ui/components/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { useRemoveOrganizationMember, useUpdateOrganizationMember } from '../api/mutations'
import { membershipExpiryInput, membershipExpiryValue } from '../lib/membership-expiry'

function memberName(member: OrganizationMember) {
  return [member.firstName, member.lastName].filter(Boolean).join(' ') || member.username || `User ${member.userId}`
}

export function MemberManagement({
  organizationId,
  members,
  roles
}: {
  organizationId: number
  members: OrganizationMember[]
  roles: RoleTemplate[]
}) {
  const [editing, setEditing] = useState<OrganizationMember | null>(null)
  const [removing, setRemoving] = useState<OrganizationMember | null>(null)
  const remove = useRemoveOrganizationMember(organizationId)
  return (
    <section className='space-y-3'>
      <h3 className='text-sm font-medium'>Manage members</h3>
      <ul className='divide-y rounded-md border'>
        {members.map((member) => (
          <li key={member.id} className='flex flex-wrap items-center justify-between gap-3 p-3'>
            <div className='min-w-0 flex-1'>
              <p className='break-words text-sm font-medium'>{memberName(member)}</p>
              <p className='text-xs text-muted-foreground'>
                {roles.find((role) => role.id === member.roleTemplateId)?.displayName || 'Unknown role'} ·{' '}
                {member.validUntil ? `Until ${new Date(member.validUntil).toLocaleString()}` : 'No expiry'}
              </p>
            </div>
            <div className='flex gap-2'>
              <Button
                size='sm'
                variant='outline'
                onClick={() => setEditing(member)}
                aria-label={`Edit membership for ${memberName(member)}`}
              >
                Edit
              </Button>
              <Button
                size='sm'
                variant='ghost'
                onClick={() => setRemoving(member)}
                aria-label={`Remove ${memberName(member)}`}
              >
                Remove
              </Button>
            </div>
          </li>
        ))}
      </ul>
      {editing && (
        <EditMembership
          key={editing.id}
          organizationId={organizationId}
          member={editing}
          roles={roles}
          onClose={() => setEditing(null)}
        />
      )}
      <AlertDialog
        open={!!removing}
        onOpenChange={(open) => {
          if (!open) setRemoving(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove {removing ? memberName(removing) : 'member'}?</AlertDialogTitle>
            <AlertDialogDescription>
              Their membership and the access it provides will be removed.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={remove.isPending}
              onClick={(event) => {
                event.preventDefault()
                if (removing)
                  remove.mutate(removing.id, {
                    onSuccess: () => {
                      setRemoving(null)
                      toast.success('Member removed')
                    },
                    onError: (error) => toast.error(error.message)
                  })
              }}
            >
              Remove member
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  )
}

function EditMembership({
  organizationId,
  member,
  roles,
  onClose
}: {
  organizationId: number
  member: OrganizationMember
  roles: RoleTemplate[]
  onClose: () => void
}) {
  const [role, setRole] = useState(String(member.roleTemplateId))
  const [expiry, setExpiry] = useState(() => membershipExpiryInput(member.validUntil))
  const update = useUpdateOrganizationMember(organizationId)
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    >
      <DialogContent className='max-w-md'>
        <DialogHeader>
          <DialogTitle>Edit membership</DialogTitle>
          <DialogDescription>{memberName(member)}</DialogDescription>
        </DialogHeader>
        <div className='space-y-2'>
          <label htmlFor='membership-role' className='text-sm font-medium'>
            Role
          </label>
          <Select
            value={role}
            onValueChange={(value) => {
              if (value) setRole(value)
            }}
          >
            <SelectTrigger id='membership-role'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {roles.map((item) => (
                <SelectItem key={item.id} value={String(item.id)}>
                  {item.displayName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className='space-y-2'>
          <label htmlFor='membership-expiry' className='text-sm font-medium'>
            Member until (optional)
          </label>
          <Input
            id='membership-expiry'
            type='datetime-local'
            value={expiry}
            onChange={(event) => setExpiry(event.target.value)}
          />
          <p className='text-xs text-muted-foreground'>Clear the date to remove the expiry.</p>
        </div>
        <DialogFooter>
          <Button variant='outline' onClick={onClose} disabled={update.isPending}>
            Cancel
          </Button>
          <Button
            disabled={update.isPending || !role}
            onClick={() => {
              try {
                update.mutate(
                  { id: member.id, roleTemplateId: Number(role), validUntil: membershipExpiryValue(expiry) },
                  {
                    onSuccess: () => {
                      toast.success('Membership updated')
                      onClose()
                    },
                    onError: (error) => toast.error(error.message)
                  }
                )
              } catch (error) {
                toast.error(error instanceof Error ? error.message : 'Invalid membership expiry')
              }
            }}
          >
            Save membership
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
