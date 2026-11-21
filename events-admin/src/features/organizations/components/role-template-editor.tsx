import { Capability, type RoleTemplate } from '@studyverse/proto'
import { ChevronRight, GripVertical, Loader2, Plus, Settings2, Shield, Trash2, Users } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { getErrorMessage } from '@/shared/lib/form-utils'
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
import { Badge } from '@/shared/ui/components/badge'
import { Button } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/shared/ui/components/dialog'
import { Input } from '@/shared/ui/components/input'
import { Label } from '@/shared/ui/components/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/components/select'
import { Switch } from '@/shared/ui/components/switch'
import { Textarea } from '@/shared/ui/components/textarea'
import { cn } from '@/shared/ui/lib/utils'
import { useCreateRoleTemplate, useDeleteRoleTemplate, useUpdateRoleTemplate } from '../api/mutations'
import { canEditRoleTemplate, getRoleUpdateInput } from '../lib/role-template-form'

interface RoleTemplateEditorProps {
  organizationId?: number
  organizationTypeId: number
  roleTemplates: RoleTemplate[]
  canManageSettings?: boolean
  triggerLabel?: string
  dialogTitle?: string
  dialogDescription?: string
}

interface RoleNodeDisplay {
  role: RoleTemplate
  children: RoleNodeDisplay[]
  depth: number
}

const CAPABILITY_LABELS: Record<Capability, { label: string; description: string }> = {
  [Capability.UNSPECIFIED]: { label: 'Unspecified', description: '' },
  [Capability.CREATE_EVENT]: { label: 'Create Events', description: 'Can create and manage events' },
  [Capability.MANAGE_MEMBERS]: { label: 'Manage Members', description: 'Can add/remove organization members' },
  [Capability.MANAGE_SETTINGS]: { label: 'Manage Settings', description: 'Can modify organization settings' },
  [Capability.VIEW_ANALYTICS]: { label: 'View Analytics', description: 'Can view statistics and reports' },
  [Capability.BOOK_ROOMS]: { label: 'Book Rooms', description: 'Can request room bookings' },
  [Capability.APPROVE_BOOKINGS]: { label: 'Approve Bookings', description: 'Can approve room booking requests' },
  [Capability.MANAGE_REGISTRATIONS]: { label: 'Manage Registrations', description: 'Can manage event registrations' },
  [Capability.CHECK_IN_ATTENDEES]: { label: 'Check-in Attendees', description: 'Can check in event attendees' }
}

const ALL_CAPABILITIES = Object.entries(CAPABILITY_LABELS)
  .filter(([key]) => key !== String(Capability.UNSPECIFIED))
  .map(([key, value]) => ({
    value: Number(key) as Capability,
    ...value
  }))

export function RoleTemplateEditor({
  organizationId,
  organizationTypeId,
  roleTemplates,
  canManageSettings = true,
  triggerLabel = 'Edit Roles',
  dialogTitle = 'Role Structure Editor',
  dialogDescription = organizationId
    ? 'Local roles apply to this organization. Shared role templates apply to all organizations of this type.'
    : 'Shared role templates apply to all organizations of this type.'
}: RoleTemplateEditorProps) {
  const [open, setOpen] = useState(false)
  const [editingRole, setEditingRole] = useState<RoleTemplate | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const { isAdmin, isStaff } = useCurrentUser()
  const canCreateRole = organizationId ? canManageSettings || isAdmin || isStaff : isAdmin || isStaff
  const selectedRoleEditable = editingRole
    ? canEditRoleTemplate(editingRole, { organizationId, canManageSettings, isAdmin, isStaff })
    : false

  return (
    <Dialog open={open} onOpenChange={(value) => !isSubmitting && setOpen(value)}>
      <DialogTrigger render={<Button variant='outline' size='sm' className='gap-2' />}>
        <Settings2 className='h-4 w-4' />
        {triggerLabel}
      </DialogTrigger>
      <DialogContent
        className='flex max-h-[85vh] w-[min(900px,calc(100vw-2rem))] flex-col overflow-hidden'
        showCloseButton={!isSubmitting}
      >
        <DialogHeader>
          <DialogTitle>{dialogTitle}</DialogTitle>
          <DialogDescription>{dialogDescription}</DialogDescription>
        </DialogHeader>

        <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto sm:flex-row sm:overflow-hidden'>
          <div className='flex max-h-64 shrink-0 flex-col overflow-hidden rounded-lg border sm:max-h-none sm:w-1/2 sm:shrink'>
            <div className='p-3 border-b bg-muted/30 flex items-center justify-between'>
              <h3 className='text-sm font-medium'>Role Hierarchy</h3>
              <Button
                size='sm'
                variant='ghost'
                className='h-8 gap-1'
                disabled={!canCreateRole || isSubmitting}
                onClick={() => {
                  setEditingRole(null)
                  setIsCreating(true)
                }}
              >
                <Plus className='h-4 w-4' />
                Add Role
              </Button>
            </div>
            <div className='flex-1 overflow-y-auto p-2'>
              <RoleHierarchyTree
                roleTemplates={roleTemplates}
                selectedRoleId={editingRole?.id}
                disabled={isSubmitting}
                onSelectRole={(role) => {
                  setEditingRole(role)
                  setIsCreating(false)
                }}
              />
            </div>
          </div>

          <div className='flex min-h-0 flex-col overflow-hidden rounded-lg border sm:w-1/2'>
            <div className='p-3 border-b bg-muted/30'>
              <h3 className='text-sm font-medium'>
                {isCreating
                  ? 'Create New Role'
                  : editingRole
                    ? selectedRoleEditable
                      ? 'Edit Role'
                      : 'View Role'
                    : 'Select a Role'}
              </h3>
            </div>
            <div className='flex-1 overflow-y-auto p-4'>
              {isCreating && canCreateRole ? (
                <CreateRoleForm
                  organizationId={organizationId}
                  organizationTypeId={organizationTypeId}
                  roleTemplates={roleTemplates}
                  onSuccess={() => {
                    setIsCreating(false)
                    setEditingRole(null)
                  }}
                  onCancel={() => setIsCreating(false)}
                  onPendingChange={setIsSubmitting}
                />
              ) : editingRole ? (
                <EditRoleForm
                  key={editingRole.id}
                  role={editingRole}
                  roleTemplates={roleTemplates}
                  organizationTypeId={organizationTypeId}
                  readOnly={!selectedRoleEditable}
                  onPendingChange={setIsSubmitting}
                  onSuccess={() => setEditingRole(null)}
                  onDelete={() => setEditingRole(null)}
                />
              ) : (
                <div className='flex flex-col items-center justify-center h-full text-muted-foreground'>
                  <Users className='h-12 w-12 mb-3 opacity-40' />
                  <p className='text-sm'>Select a role to edit or create a new one</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

interface RoleHierarchyTreeProps {
  roleTemplates: RoleTemplate[]
  selectedRoleId?: number
  onSelectRole: (role: RoleTemplate) => void
  disabled?: boolean
}

function RoleHierarchyTree({ roleTemplates, selectedRoleId, onSelectRole, disabled }: RoleHierarchyTreeProps) {
  const hierarchy = useMemo(() => {
    const rootRoles = roleTemplates.filter((r) => !r.parentTemplateId).sort((a, b) => a.sortOrder - b.sortOrder)

    const childrenByParent = new Map<number, RoleTemplate[]>()
    for (const role of roleTemplates) {
      if (role.parentTemplateId) {
        const siblings = childrenByParent.get(role.parentTemplateId) || []
        siblings.push(role)
        childrenByParent.set(role.parentTemplateId, siblings)
      }
    }

    function buildTree(role: RoleTemplate, depth: number): RoleNodeDisplay {
      const children = (childrenByParent.get(role.id) || [])
        .sort((a, b) => a.sortOrder - b.sortOrder)
        .map((child) => buildTree(child, depth + 1))

      return { role, children, depth }
    }

    return rootRoles.map((role) => buildTree(role, 0))
  }, [roleTemplates])

  if (roleTemplates.length === 0) {
    return (
      <div className='flex flex-col items-center justify-center py-8 text-muted-foreground'>
        <Shield className='h-10 w-10 mb-2 opacity-40' />
        <p className='text-sm'>No roles defined yet</p>
        <p className='text-xs'>Create roles to build your hierarchy</p>
      </div>
    )
  }

  return (
    <div className='space-y-1'>
      {hierarchy.map((node) => (
        <RoleTreeNode
          key={node.role.id}
          node={node}
          selectedRoleId={selectedRoleId}
          onSelectRole={onSelectRole}
          disabled={disabled}
        />
      ))}
    </div>
  )
}

interface RoleTreeNodeProps {
  node: RoleNodeDisplay
  selectedRoleId?: number
  onSelectRole: (role: RoleTemplate) => void
  disabled?: boolean
}

function RoleTreeNode({ node, selectedRoleId, onSelectRole, disabled }: RoleTreeNodeProps) {
  const [expanded, setExpanded] = useState(true)
  const hasChildren = node.children.length > 0
  const isSelected = selectedRoleId === node.role.id

  return (
    <div>
      <div
        className={cn(
          'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors',
          isSelected ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
        )}
        style={{ paddingLeft: `${node.depth * 16 + 8}px` }}
      >
        {hasChildren ? (
          <button
            type='button'
            disabled={disabled}
            aria-label={`${expanded ? 'Collapse' : 'Expand'} ${node.role.displayName}`}
            aria-expanded={expanded}
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className='p-0.5 hover:bg-black/10 rounded'
          >
            <ChevronRight className={cn('h-4 w-4 transition-transform', expanded && 'rotate-90')} />
          </button>
        ) : (
          <span className='w-5' />
        )}
        <GripVertical className='h-4 w-4 opacity-40' />
        <button
          type='button'
          disabled={disabled}
          onClick={() => onSelectRole(node.role)}
          className='min-w-0 flex-1 text-left'
        >
          <span className='block truncate'>{node.role.displayName}</span>
          <span className='block text-[10px] opacity-70'>
            {node.role.organizationId ? 'Local role' : 'Shared role template'}
            {!node.role.isActive && ' · Inactive'}
          </span>
        </button>
        {node.role.maxHolders && (
          <Badge variant='outline' className={cn('text-xs', isSelected && 'border-primary-foreground/30')}>
            max {node.role.maxHolders}
          </Badge>
        )}
        {node.role.isRequired && (
          <Badge variant='secondary' className={cn('text-xs', isSelected && 'bg-primary-foreground/20')}>
            required
          </Badge>
        )}
      </div>
      {hasChildren && expanded && (
        <div className='mt-1'>
          {node.children.map((child) => (
            <RoleTreeNode
              key={child.role.id}
              node={child}
              selectedRoleId={selectedRoleId}
              onSelectRole={onSelectRole}
              disabled={disabled}
            />
          ))}
        </div>
      )}
    </div>
  )
}

interface CreateRoleFormProps {
  organizationId?: number
  organizationTypeId: number
  roleTemplates: RoleTemplate[]
  onSuccess: () => void
  onCancel: () => void
  onPendingChange: (pending: boolean) => void
}

function CreateRoleForm({
  organizationId,
  organizationTypeId,
  roleTemplates,
  onSuccess,
  onCancel,
  onPendingChange
}: CreateRoleFormProps) {
  const [displayName, setDisplayName] = useState('')
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [parentTemplateId, setParentTemplateId] = useState<string>('')
  const [maxHolders, setMaxHolders] = useState<string>('')
  const [isRequired, setIsRequired] = useState(false)
  const [isActive, setIsActive] = useState(true)
  const [sortOrder, setSortOrder] = useState(String(roleTemplates.length))
  const [validationError, setValidationError] = useState('')
  const [selectedCapabilities, setSelectedCapabilities] = useState<Capability[]>([])

  const createMutation = useCreateRoleTemplate(organizationTypeId, organizationId)
  useEffect(() => onPendingChange(createMutation.isPending), [createMutation.isPending, onPendingChange])
  useEffect(() => () => onPendingChange(false), [onPendingChange])

  const sortedRoles = [...roleTemplates].sort((a, b) => a.sortOrder - b.sortOrder)

  const toggleCapability = (cap: Capability) => {
    setSelectedCapabilities((prev) => (prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap]))
  }

  const handleCreate = () => {
    setValidationError('')
    let input: ReturnType<typeof getRoleUpdateInput>
    try {
      input = getRoleUpdateInput({
        id: 0,
        displayName,
        description,
        parentTemplateId,
        sortOrder,
        maxHolders,
        isRequired,
        isActive,
        capabilities: selectedCapabilities
      })
    } catch (error) {
      setValidationError(getErrorMessage(error))
      return
    }
    createMutation.mutate(
      {
        organizationId: organizationId || undefined,
        key: key.trim() || input.displayName.toLowerCase().replace(/\s+/g, '_'),
        displayName: input.displayName,
        description: input.description,
        parentTemplateId: input.parentTemplateId,
        sortOrder: input.sortOrder,
        maxHolders: input.maxHolders,
        isRequired: input.isRequired,
        isActive: input.isActive,
        capabilities: input.capabilities
      },
      {
        onSuccess: () => {
          toast.success('Role created successfully')
          onSuccess()
        },
        onError: (error) => {
          toast.error(`Failed to create role: ${error.message}`)
        }
      }
    )
  }

  return (
    <div className='space-y-4'>
      <RoleScopeNotice organizationId={organizationId} />
      <fieldset disabled={createMutation.isPending} className='space-y-4'>
        <legend className='sr-only'>New role settings</legend>
        <div className='space-y-2'>
          <Label htmlFor='displayName'>Display Name *</Label>
          <Input
            id='displayName'
            placeholder='e.g., Vice President of Science'
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='key'>Key (optional)</Label>
          <Input
            id='key'
            placeholder='e.g., vp_science (auto-generated if empty)'
            value={key}
            onChange={(e) => setKey(e.target.value)}
          />
          <p className='text-xs text-muted-foreground'>
            Unique identifier for this role. Auto-generated from display name if left empty.
          </p>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='description'>Description</Label>
          <Textarea
            id='description'
            placeholder='Describe the responsibilities of this role...'
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='parentRole'>Reports To</Label>
          <Select
            disabled={createMutation.isPending}
            value={parentTemplateId || 'none'}
            onValueChange={(v) => {
              if (v === null) return
              return setParentTemplateId(v === 'none' ? '' : v)
            }}
          >
            <SelectTrigger id='parentRole'>
              <SelectValue placeholder='Select parent role (optional)' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='none'>No parent (top level)</SelectItem>
              {sortedRoles.map((role) => (
                <SelectItem key={role.id} value={role.id.toString()}>
                  {role.displayName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='maxHolders'>Max Members</Label>
            <Input
              id='maxHolders'
              type='number'
              min='1'
              placeholder='Unlimited'
              value={maxHolders}
              onChange={(e) => setMaxHolders(e.target.value)}
            />
          </div>

          <div className='flex items-center gap-2 pt-6'>
            <Switch
              id='isRequired'
              checked={isRequired}
              onCheckedChange={setIsRequired}
              disabled={createMutation.isPending}
            />
            <Label htmlFor='isRequired'>Required role</Label>
          </div>
        </div>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='sortOrder'>Sort Order</Label>
            <Input
              id='sortOrder'
              type='number'
              step='1'
              value={sortOrder}
              onChange={(event) => setSortOrder(event.target.value)}
            />
          </div>
          <div className='flex items-center gap-2 pt-6'>
            <Switch
              id='isActive'
              checked={isActive}
              onCheckedChange={setIsActive}
              disabled={createMutation.isPending}
            />
            <Label htmlFor='isActive'>Active role</Label>
          </div>
        </div>

        <div className='space-y-2'>
          <Label>Capabilities</Label>
          <div className='grid grid-cols-1 gap-2 max-h-40 overflow-y-auto border rounded-md p-3'>
            {ALL_CAPABILITIES.map((cap) => (
              <div key={cap.value} className='flex items-start gap-2'>
                <Checkbox
                  id={`cap-${cap.value}`}
                  disabled={createMutation.isPending}
                  checked={selectedCapabilities.includes(cap.value)}
                  onCheckedChange={() => toggleCapability(cap.value)}
                />
                <div className='grid gap-0.5 leading-none'>
                  <label htmlFor={`cap-${cap.value}`} className='text-sm font-medium cursor-pointer'>
                    {cap.label}
                  </label>
                  <p className='text-xs text-muted-foreground'>{cap.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </fieldset>

      {(validationError || createMutation.error) && (
        <p role='alert' className='text-sm text-destructive'>
          {validationError || createMutation.error?.message}
        </p>
      )}

      <div className='flex justify-end gap-2 pt-2'>
        <Button variant='outline' onClick={onCancel} disabled={createMutation.isPending}>
          Cancel
        </Button>
        <Button onClick={handleCreate} disabled={!displayName.trim() || createMutation.isPending}>
          {createMutation.isPending && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
          Create Role
        </Button>
      </div>
    </div>
  )
}

interface EditRoleFormProps {
  role: RoleTemplate
  roleTemplates: RoleTemplate[]
  organizationTypeId: number
  onSuccess: () => void
  onDelete: () => void
  readOnly: boolean
  onPendingChange: (pending: boolean) => void
}

function EditRoleForm({
  role,
  roleTemplates,
  organizationTypeId,
  onSuccess,
  onDelete,
  readOnly,
  onPendingChange
}: EditRoleFormProps) {
  const [displayName, setDisplayName] = useState(role.displayName)
  const [description, setDescription] = useState(role.description || '')
  const [parentTemplateId, setParentTemplateId] = useState<string>(role.parentTemplateId?.toString() || '')
  const [maxHolders, setMaxHolders] = useState<string>(role.maxHolders?.toString() || '')
  const [isRequired, setIsRequired] = useState(role.isRequired)
  const [isActive, setIsActive] = useState(role.isActive)
  const [sortOrder, setSortOrder] = useState(String(role.sortOrder))
  const [validationError, setValidationError] = useState('')
  const [selectedCapabilities, setSelectedCapabilities] = useState<Capability[]>(role.capabilities || [])

  const updateMutation = useUpdateRoleTemplate(organizationTypeId, role.organizationId)
  const deleteMutation = useDeleteRoleTemplate(organizationTypeId, role.organizationId)
  const isPending = updateMutation.isPending || deleteMutation.isPending
  const disabled = readOnly || isPending
  const [deleteOpen, setDeleteOpen] = useState(false)
  useEffect(() => onPendingChange(isPending), [isPending, onPendingChange])
  useEffect(() => () => onPendingChange(false), [onPendingChange])

  const availableParents = roleTemplates.filter((r) => {
    if (r.id === role.id) return false
    if (!role.organizationId && r.organizationId) return false
    if (r.organizationId && r.organizationId !== role.organizationId) return false
    let current: RoleTemplate | undefined = r
    const visited = new Set<number>()
    while (current) {
      if (visited.has(current.id)) return false
      visited.add(current.id)
      if (current.parentTemplateId === role.id) return false
      current = roleTemplates.find((rt) => rt.id === current?.parentTemplateId)
    }
    return true
  })

  const sortedParents = [...availableParents].sort((a, b) => a.sortOrder - b.sortOrder)

  const toggleCapability = (cap: Capability) => {
    setSelectedCapabilities((prev) => (prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap]))
  }

  const handleSave = () => {
    if (readOnly || isPending) return
    setValidationError('')
    let input: ReturnType<typeof getRoleUpdateInput>
    try {
      input = getRoleUpdateInput({
        id: role.id,
        displayName,
        description,
        parentTemplateId,
        maxHolders,
        sortOrder,
        isRequired,
        isActive,
        capabilities: selectedCapabilities
      })
    } catch (error) {
      setValidationError(getErrorMessage(error))
      return
    }
    updateMutation.mutate(input, {
      onSuccess: () => {
        toast.success('Role updated successfully')
        onSuccess()
      },
      onError: (error) => toast.error(`Failed to update role: ${error.message}`)
    })
  }

  return (
    <div className='space-y-4'>
      <RoleScopeNotice organizationId={role.organizationId} readOnly={readOnly} />
      <div className='flex items-center gap-2 pb-2 border-b'>
        <Badge variant='outline' className='font-mono text-xs'>
          {role.key}
        </Badge>
        <span className='text-xs text-muted-foreground'>ID: {role.id}</span>
      </div>

      <fieldset disabled={disabled} className='space-y-4'>
        <legend className='sr-only'>Role settings</legend>
        <div className='space-y-2'>
          <Label htmlFor='editDisplayName'>Display Name</Label>
          <Input id='editDisplayName' value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='editDescription'>Description</Label>
          <Textarea
            id='editDescription'
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='editParentRole'>Reports To</Label>
          <Select
            disabled={disabled}
            value={parentTemplateId || 'none'}
            onValueChange={(v) => {
              if (v === null) return
              return setParentTemplateId(v === 'none' ? '' : v)
            }}
          >
            <SelectTrigger id='editParentRole'>
              <SelectValue placeholder='No parent (top level)' />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='none'>No parent (top level)</SelectItem>
              {sortedParents.map((r) => (
                <SelectItem key={r.id} value={r.id.toString()}>
                  {r.displayName}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='editMaxHolders'>Max Members</Label>
            <Input
              id='editMaxHolders'
              type='number'
              min='1'
              placeholder='Unlimited'
              value={maxHolders}
              onChange={(e) => setMaxHolders(e.target.value)}
            />
          </div>

          <div className='flex items-center gap-2 pt-6'>
            <Switch id='editIsRequired' checked={isRequired} onCheckedChange={setIsRequired} disabled={disabled} />
            <Label htmlFor='editIsRequired'>Required role</Label>
          </div>
        </div>

        <div className='grid grid-cols-2 gap-4'>
          <div className='space-y-2'>
            <Label htmlFor='editSortOrder'>Sort Order</Label>
            <Input
              id='editSortOrder'
              type='number'
              step='1'
              value={sortOrder}
              onChange={(event) => setSortOrder(event.target.value)}
            />
          </div>
          <div className='flex items-center gap-2 pt-6'>
            <Switch id='editIsActive' checked={isActive} onCheckedChange={setIsActive} disabled={disabled} />
            <Label htmlFor='editIsActive'>Active role</Label>
          </div>
        </div>

        <div className='space-y-2'>
          <Label>Capabilities</Label>
          <div className='grid grid-cols-1 gap-2 max-h-40 overflow-y-auto border rounded-md p-3'>
            {ALL_CAPABILITIES.map((cap) => (
              <div key={cap.value} className='flex items-start gap-2'>
                <Checkbox
                  id={`edit-cap-${cap.value}`}
                  disabled={disabled}
                  checked={selectedCapabilities.includes(cap.value)}
                  onCheckedChange={() => toggleCapability(cap.value)}
                />
                <div className='grid gap-0.5 leading-none'>
                  <label htmlFor={`edit-cap-${cap.value}`} className='text-sm font-medium cursor-pointer'>
                    {cap.label}
                  </label>
                  <p className='text-xs text-muted-foreground'>{cap.description}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </fieldset>

      {(validationError || updateMutation.error || deleteMutation.error) && (
        <p role='alert' className='text-sm text-destructive'>
          {validationError || updateMutation.error?.message || deleteMutation.error?.message}
        </p>
      )}

      {!readOnly && (
        <div className='flex justify-between gap-2 pt-2'>
          <AlertDialog open={deleteOpen} onOpenChange={(value) => !isPending && setDeleteOpen(value)}>
            <AlertDialogTrigger render={<Button variant='destructive' size='sm' disabled={isPending} />}>
              <Trash2 className='h-4 w-4 mr-1' />
              Delete
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Delete Role</AlertDialogTitle>
                <AlertDialogDescription>
                  Are you sure you want to delete "{role.displayName}"? This action cannot be undone. Members with this
                  role will need to be reassigned.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel disabled={isPending}>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  disabled={isPending}
                  onClick={(event) => {
                    event.preventDefault()
                    deleteMutation.mutate(role.id, {
                      onSuccess: () => {
                        setDeleteOpen(false)
                        toast.success('Role deleted successfully')
                        onDelete()
                      },
                      onError: (error) => {
                        toast.error(`Failed to delete role: ${error.message}`)
                      }
                    })
                  }}
                  className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                >
                  {deleteMutation.isPending ? <Loader2 className='h-4 w-4 animate-spin' /> : 'Delete'}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>

          <Button onClick={handleSave} disabled={!displayName.trim() || isPending}>
            {updateMutation.isPending && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            Save Changes
          </Button>
        </div>
      )}
    </div>
  )
}

function RoleScopeNotice({ organizationId, readOnly }: { organizationId?: number; readOnly?: boolean }) {
  return (
    <div className='space-y-1 rounded-md border px-3 py-2'>
      <p className='text-xs font-medium'>{organizationId ? 'Local role' : 'Shared role template'}</p>
      <p className='text-xs text-muted-foreground'>
        {organizationId ? 'Applies to this organization only.' : 'Applies to all organizations of this type.'}
        {readOnly &&
          (!organizationId ? ' Only platform admins and staff can edit this template.' : ' You have view-only access.')}
      </p>
    </div>
  )
}
