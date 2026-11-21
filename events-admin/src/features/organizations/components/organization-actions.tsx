import type { Organization } from '@studyverse/proto'
import { Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger
} from '@/shared/ui/components/alert-dialog'
import { Button } from '@/shared/ui/components/button'
import { useDeleteOrganization } from '../api/mutations'
import { CreateOrganizationForm } from './create-organization-form'

export function OrganizationActions({
  organization,
  canEdit,
  onEdit,
  onDeleted
}: {
  organization: Organization
  canEdit: boolean
  onEdit?: () => void
  onDeleted: () => void
}) {
  const [editing, setEditing] = useState(false)
  const { isAdmin, isStaff } = useCurrentUser()
  const deletion = useDeleteOrganization()
  return (
    <div className='flex items-center gap-2'>
      {canEdit && (
        <Button variant='outline' size='sm' onClick={onEdit ?? (() => setEditing(true))}>
          <Pencil className='mr-2 size-4' />
          Edit club
        </Button>
      )}
      {(isAdmin || isStaff) && (
        <AlertDialog>
          <AlertDialogTrigger render={<Button variant='ghost' size='sm' aria-label='Delete club' />}>
            <Trash2 className='size-4' />
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete {organization.title}?</AlertDialogTitle>
              <AlertDialogDescription>
                This permanently deletes the organization. Archive it in Edit club if you want to keep its records.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deletion.isPending}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                disabled={deletion.isPending}
                onClick={(event) => {
                  event.preventDefault()
                  deletion.mutate(
                    { id: organization.id },
                    {
                      onSuccess: () => {
                        toast.success('Organization deleted')
                        onDeleted()
                      },
                      onError: (error) => toast.error(error.message)
                    }
                  )
                }}
              >
                Delete organization
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
      {editing && (
        <CreateOrganizationForm
          key={organization.updatedAt}
          open={editing}
          onOpenChange={setEditing}
          organizationId={organization.id}
          existingImageUrl={organization.imageUrl}
          initialValues={{
            title: organization.title,
            description: organization.description || '',
            organizationTypeId: organization.organizationTypeId,
            status: organization.status,
            instagram: organization.instagram || '',
            website: organization.website || '',
            telegramChannel: organization.telegramChannel || '',
            telegramChat: organization.telegramChat || '',
            youtube: organization.youtube || '',
            tiktok: organization.tiktok || '',
            linkedin: organization.linkedin || ''
          }}
        />
      )}
    </div>
  )
}
