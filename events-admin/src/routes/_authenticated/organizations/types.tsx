import type { OrganizationType } from '@studyverse/proto'
import { createFileRoute, Link } from '@tanstack/react-router'
import { ArrowLeft, Building2, Loader2, Pencil, Plus, Trash2 } from 'lucide-react'
import { Suspense, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  RoleTemplateEditor,
  useCreateOrganizationType,
  useDeleteOrganizationType,
  useOrganizations,
  useOrganizationTypes,
  useRoleTemplatesByOrgTypeQuery,
  useUpdateOrganizationType
} from '@/features/organizations'
import { useAuth } from '@/shared/lib/auth'
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
import { Badge } from '@/shared/ui/components/badge'
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
import { Skeleton } from '@/shared/ui/components/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/shared/ui/components/table'

export const Route = createFileRoute('/_authenticated/organizations/types')({
  component: () => (
    <Suspense fallback={<OrganizationTypesLoading />}>
      <OrganizationTypesPage />
    </Suspense>
  )
})

function OrganizationTypesLoading() {
  return (
    <div className='space-y-6'>
      <div className='flex items-center gap-4'>
        <Skeleton className='h-10 w-10' />
        <div className='space-y-2'>
          <Skeleton className='h-8 w-48' />
          <Skeleton className='h-5 w-64' />
        </div>
      </div>
      <div className='rounded-lg border'>
        {['type-row-1', 'type-row-2', 'type-row-3', 'type-row-4', 'type-row-5'].map((key) => (
          <div key={key} className='border-b px-4 py-3 last:border-0'>
            <Skeleton className='h-5 w-full max-w-xs' />
          </div>
        ))}
      </div>
    </div>
  )
}

function OrganizationTypesPage() {
  const { isGuest } = useAuth()
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingType, setEditingType] = useState<OrganizationType | null>(null)
  const [newTitle, setNewTitle] = useState('')
  const [editTitle, setEditTitle] = useState('')

  const { data: typesData } = useOrganizationTypes({})
  const { data: orgsData } = useOrganizations({ limit: 1000, page: 1 })

  const createMutation = useCreateOrganizationType()
  const updateMutation = useUpdateOrganizationType()
  const deleteMutation = useDeleteOrganizationType()

  const handleCreate = () => {
    if (!newTitle.trim()) return
    createMutation.mutate(newTitle.trim(), {
      onSuccess: () => {
        setIsCreateOpen(false)
        setNewTitle('')
        toast.success('Organization type created')
      },
      onError: (error: unknown) => {
        toast.error('Failed to create organization type', {
          description: error instanceof Error ? error.message : 'Unknown error'
        })
      }
    })
  }

  const handleUpdate = () => {
    if (!editingType || !editTitle.trim()) return
    updateMutation.mutate(
      { id: editingType.id, title: editTitle.trim() },
      {
        onSuccess: () => {
          setEditingType(null)
          setEditTitle('')
          toast.success('Organization type updated')
        },
        onError: (error: unknown) => {
          toast.error('Failed to update organization type', {
            description: error instanceof Error ? error.message : 'Unknown error'
          })
        }
      }
    )
  }

  const handleEdit = (type: OrganizationType) => {
    setEditingType(type)
    setEditTitle(type.title)
  }

  const orgCountByType = useMemo(() => {
    const counts: Record<number, number> = {}
    orgsData.organizations.forEach((org: { organizationTypeId: number }) => {
      counts[org.organizationTypeId] = (counts[org.organizationTypeId] || 0) + 1
    })
    return counts
  }, [orgsData.organizations])

  return (
    <div className='space-y-6'>
      <div className='flex items-start gap-4'>
        <Button variant='ghost' size='icon' className='mt-1' render={<Link to='/organizations' />} nativeButton={false}>
          <ArrowLeft className='h-4 w-4' />
        </Button>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>Organization Types</h1>
          <p className='text-sm text-muted-foreground mt-1'>
            Manage organization types and their default role templates
          </p>
        </div>
      </div>

      <div className='flex justify-end'>
        {!isGuest && (
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className='h-4 w-4 mr-2' />
            Add Type
          </Button>
        )}
      </div>

      {typesData.organizationTypes.length === 0 ? (
        <div className='flex flex-col items-center justify-center gap-4 py-12 text-muted-foreground border rounded-lg'>
          <Building2 className='h-12 w-12 opacity-40' />
          <div className='text-center'>
            <p className='font-medium'>No organization types</p>
            <p className='text-sm'>Create your first organization type to get started</p>
          </div>
        </div>
      ) : (
        <div className='rounded-lg border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead className='w-[120px]'>Organizations</TableHead>
                <TableHead className='w-[140px]'>Default Roles</TableHead>
                <TableHead className='w-[100px]'>Created</TableHead>
                {!isGuest && <TableHead className='w-[100px] text-right'>Actions</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {typesData.organizationTypes.map((type: OrganizationType) => {
                const count = orgCountByType[type.id] || 0
                return (
                  <TableRow key={type.id}>
                    <TableCell className='font-medium'>{type.title}</TableCell>
                    <TableCell>
                      <Badge variant='secondary' className='font-normal'>
                        {count} {count === 1 ? 'org' : 'orgs'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <TypeRolesButton typeId={type.id} typeTitle={type.title} />
                    </TableCell>
                    <TableCell className='text-muted-foreground text-sm'>
                      {new Date(type.createdAt).toLocaleDateString()}
                    </TableCell>
                    {!isGuest && (
                      <TableCell className='text-right'>
                        <div className='flex items-center justify-end gap-1'>
                          <Button variant='ghost' size='icon' className='h-8 w-8' onClick={() => handleEdit(type)}>
                            <Pencil className='h-4 w-4' />
                          </Button>
                          <AlertDialog>
                            <AlertDialogTrigger
                              render={<Button variant='ghost' size='icon' className='h-8 w-8 text-destructive' />}
                            >
                              <Trash2 className='h-4 w-4' />
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>Delete organization type?</AlertDialogTitle>
                                <AlertDialogDescription>
                                  This will permanently delete "{type.title}". Organizations using this type may be
                                  affected.
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() =>
                                    deleteMutation.mutate(type.id, {
                                      onSuccess: () => {
                                        toast.success('Organization type deleted')
                                      },
                                      onError: (error: unknown) => {
                                        toast.error('Failed to delete organization type', {
                                          description: error instanceof Error ? error.message : 'Unknown error'
                                        })
                                      }
                                    })
                                  }
                                  className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                                >
                                  {deleteMutation.isPending ? <Loader2 className='h-4 w-4 animate-spin' /> : 'Delete'}
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        </div>
                      </TableCell>
                    )}
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create Organization Type</DialogTitle>
            <DialogDescription>
              Add a new type for categorizing organizations (e.g., Club, Department, Committee)
            </DialogDescription>
          </DialogHeader>
          <div className='py-4'>
            <Input
              placeholder='Type title (e.g., Student Club)'
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
            />
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setIsCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={!newTitle.trim() || createMutation.isPending}>
              {createMutation.isPending && <Loader2 className='h-4 w-4 animate-spin mr-2' />}
              Create
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editingType} onOpenChange={(open) => !open && setEditingType(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Organization Type</DialogTitle>
            <DialogDescription>Update the organization type title</DialogDescription>
          </DialogHeader>
          <div className='py-4'>
            <Input
              placeholder='Type title'
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleUpdate()}
            />
          </div>
          <DialogFooter>
            <Button variant='outline' onClick={() => setEditingType(null)}>
              Cancel
            </Button>
            <Button onClick={handleUpdate} disabled={!editTitle.trim() || updateMutation.isPending}>
              {updateMutation.isPending && <Loader2 className='h-4 w-4 animate-spin mr-2' />}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function TypeRolesButton({ typeId, typeTitle }: { typeId: number; typeTitle: string }) {
  const { data, isLoading } = useRoleTemplatesByOrgTypeQuery({ organizationTypeId: typeId })

  if (isLoading) {
    return (
      <Button variant='ghost' size='icon' className='h-8 w-8' disabled>
        <Loader2 className='h-4 w-4 animate-spin' />
      </Button>
    )
  }

  return (
    <RoleTemplateEditor
      organizationTypeId={typeId}
      roleTemplates={data?.roleTemplates || []}
      triggerLabel='Default Roles'
      dialogTitle={`Default Roles for ${typeTitle}`}
      dialogDescription='Define the default role hierarchy for this organization type. New organizations of this type will inherit these roles.'
    />
  )
}
