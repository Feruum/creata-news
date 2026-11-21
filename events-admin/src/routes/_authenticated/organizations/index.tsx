import type { Organization, OrganizationType } from '@studyverse/proto'
import { OrganizationStatus } from '@studyverse/proto'
import { createFileRoute } from '@tanstack/react-router'
import { Building2, FilterIcon, PlusIcon } from 'lucide-react'
import { parseAsArrayOf, parseAsInteger, useQueryState } from 'nuqs'
import { Suspense, useCallback, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  CreateOrganizationForm,
  columns,
  OrganizationInfoModalControlled,
  useDeleteOrganization,
  useOrganizations,
  useOrganizationTypes
} from '@/features/organizations'
import { DataTable, useDataTableState } from '@/shared/components/admin-data-table'
import { useAuth } from '@/shared/lib/auth'
import { useCurrentUser } from '@/shared/lib/use-current-user'
import { Button } from '@/shared/ui/components/button'
import { Checkbox } from '@/shared/ui/components/checkbox'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/components/popover'
import { Skeleton } from '@/shared/ui/components/skeleton'

export const Route = createFileRoute('/_authenticated/organizations/')({
  component: () => (
    <Suspense fallback={<OrganizationsLoading />}>
      <Organizations />
    </Suspense>
  )
})

const organizationLoadingRowIds = [
  'org-row-1',
  'org-row-2',
  'org-row-3',
  'org-row-4',
  'org-row-5',
  'org-row-6',
  'org-row-7',
  'org-row-8',
  'org-row-9',
  'org-row-10'
]

function OrganizationsLoading() {
  return (
    <div className='space-y-4'>
      {/* Header */}
      <div className='mb-6 space-y-2'>
        <Skeleton className='h-8 w-40' />
        <Skeleton className='h-5 w-48' />
      </div>

      {/* Toolbar */}
      <div className='mb-4 flex items-center justify-between gap-3'>
        <Skeleton className='h-10 w-64' />
        <Skeleton className='h-10 w-36' />
      </div>

      {/* Table */}
      <div className='rounded-lg border'>
        <div className='border-b px-4 py-3'>
          <Skeleton className='h-4 w-full max-w-2xl' />
        </div>
        {organizationLoadingRowIds.map((rowId) => (
          <div key={rowId} className='border-b px-4 py-3 last:border-0'>
            <Skeleton className='h-5 w-full' />
          </div>
        ))}
      </div>

      {/* Pagination */}
      <div className='mt-4 flex items-center justify-between'>
        <Skeleton className='h-9 w-28' />
        <Skeleton className='h-5 w-40' />
        <Skeleton className='h-9 w-24' />
      </div>
    </div>
  )
}

// Status filter options
const statusOptions = [
  { value: OrganizationStatus.ACTIVE, label: 'Active' },
  { value: OrganizationStatus.ARCHIVED, label: 'Archived' },
  { value: OrganizationStatus.FROZEN, label: 'Frozen' }
]

function Organizations() {
  const { isGuest } = useAuth()
  const { isAdmin, isStaff } = useCurrentUser()
  const canDelete = !isGuest && (isAdmin || isStaff)
  const deleteMutation = useDeleteOrganization()
  const { tableState } = useDataTableState({ defaultSortBy: 'title', defaultSortDesc: false })
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [selectedOrgId, setSelectedOrgId] = useState<number | null>(null)

  const [statusFilter, setStatusFilter] = useQueryState('status', parseAsArrayOf(parseAsInteger).withDefault([]))
  const [typeFilter, setTypeFilter] = useQueryState('type', parseAsArrayOf(parseAsInteger).withDefault([]))

  const hasActiveFilters = Boolean(tableState.search) || statusFilter.length > 0 || typeFilter.length > 0

  const { data } = useOrganizations({
    limit: hasActiveFilters ? 1000 : tableState.pageSize,
    page: hasActiveFilters ? 1 : tableState.page
  })

  const { data: typesData } = useOrganizationTypes({})

  const handleStatusChange = (checked: boolean, value: number) => {
    const newValues = statusFilter ? [...statusFilter] : []
    if (checked) {
      newValues.push(value)
    } else {
      const index = newValues.indexOf(value)
      if (index > -1) {
        newValues.splice(index, 1)
      }
    }
    setStatusFilter(newValues.length ? newValues : null)
  }

  const handleTypeChange = (checked: boolean, value: number) => {
    const newValues = typeFilter ? [...typeFilter] : []
    if (checked) {
      newValues.push(value)
    } else {
      const index = newValues.indexOf(value)
      if (index > -1) {
        newValues.splice(index, 1)
      }
    }
    setTypeFilter(newValues.length ? newValues : null)
  }

  const handleRowClick = useCallback((org: Organization) => {
    setSelectedOrgId(org.id)
  }, [])

  const handleDeleteSelected = useCallback(
    async (ids: (string | number)[]) => {
      if (!canDelete || deleteMutation.isPending) return
      try {
        for (const id of ids) {
          await deleteMutation.mutateAsync({ id: Number(id) })
        }
        toast.success(ids.length === 1 ? 'Organization deleted' : `${ids.length} organizations deleted`)
      } catch {
        toast.error('Failed to delete organization')
      }
    },
    [canDelete, deleteMutation]
  )

  const filteredOrganizations = useMemo(() => {
    let result = data.organizations
    if (statusFilter.length > 0) {
      result = result.filter((org: Organization) => statusFilter.includes(org.status))
    }
    if (typeFilter.length > 0) {
      result = result.filter((org: Organization) => typeFilter.includes(org.organizationTypeId))
    }
    return result
  }, [data.organizations, statusFilter, typeFilter])

  return (
    <div className='space-y-6'>
      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Organizations</h1>
        <p className='text-sm text-muted-foreground mt-1'>Manage organizations</p>
      </div>

      <DataTable
        data={filteredOrganizations}
        columns={columns}
        totalCount={statusFilter.length > 0 || typeFilter.length > 0 ? filteredOrganizations.length : data.total || 0}
        hasActiveFilters={hasActiveFilters}
        searchColumnId='title'
        searchPlaceholder='Search organizations...'
        entityName='organizations'
        defaultSortBy='title'
        defaultSortDesc={false}
        readOnly={!canDelete}
        onDeleteSelected={canDelete ? handleDeleteSelected : undefined}
        isDeleting={deleteMutation.isPending}
        onRowClick={handleRowClick}
        activeRowId={selectedOrgId === null ? null : String(selectedOrgId)}
        filterComponents={
          <>
            <StatusFilter
              statusOptions={statusOptions}
              selectedValues={statusFilter}
              onFilterChange={handleStatusChange}
            />
            <TypeFilter
              types={typesData.organizationTypes}
              selectedValues={typeFilter}
              onFilterChange={handleTypeChange}
            />
          </>
        }
        toolbarActions={
          !isGuest && (
            <Button className='ml-auto' onClick={() => setIsCreateOpen(true)}>
              <PlusIcon aria-hidden='true' className='-ms-1 opacity-60' size={16} />
              Add organization
            </Button>
          )
        }
      />

      <Suspense fallback={null}>
        <CreateOrganizationForm open={isCreateOpen} onOpenChange={setIsCreateOpen} />
      </Suspense>

      <OrganizationInfoModalControlled
        organizationId={selectedOrgId}
        onOrganizationChange={setSelectedOrgId}
        onClose={() => setSelectedOrgId(null)}
      />
    </div>
  )
}

// Status filter component
function StatusFilter({
  statusOptions,
  selectedValues,
  onFilterChange
}: {
  statusOptions: { value: number; label: string }[]
  selectedValues: number[]
  onFilterChange: (checked: boolean, value: number) => void
}) {
  return (
    <Popover>
      <PopoverTrigger render={<Button variant='outline' />}>
        <FilterIcon aria-hidden='true' className='-ms-1 opacity-60' size={16} />
        Status
        {selectedValues.length > 0 && (
          <span className='-me-1 inline-flex h-5 max-h-full items-center rounded border bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-muted-foreground/70'>
            {selectedValues.length}
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align='start' className='w-auto min-w-36 p-3'>
        <div className='space-y-3'>
          <div className='text-xs font-medium text-muted-foreground'>Filter by status</div>
          <div className='space-y-3'>
            {statusOptions.map((option) => {
              const checkboxId = `status-filter-${option.value}`
              return (
                <div key={option.value} className='flex items-center gap-2'>
                  <Checkbox
                    id={checkboxId}
                    checked={selectedValues.includes(option.value)}
                    onCheckedChange={(checked) => onFilterChange(!!checked, option.value)}
                  />
                  <label htmlFor={checkboxId} className='cursor-pointer text-sm'>
                    {option.label}
                  </label>
                </div>
              )
            })}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function TypeFilter({
  types,
  selectedValues,
  onFilterChange
}: {
  types: OrganizationType[]
  selectedValues: number[]
  onFilterChange: (checked: boolean, value: number) => void
}) {
  return (
    <Popover>
      <PopoverTrigger render={<Button variant='outline' />}>
        <Building2 aria-hidden='true' className='-ms-1 opacity-60' size={16} />
        Type
        {selectedValues.length > 0 && (
          <span className='-me-1 inline-flex h-5 max-h-full items-center rounded border bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-muted-foreground/70'>
            {selectedValues.length}
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align='start' className='w-auto min-w-36 p-3'>
        <div className='space-y-3'>
          <div className='text-xs font-medium text-muted-foreground'>Filter by type</div>
          <div className='space-y-3'>
            {types.map((type) => {
              const checkboxId = `type-filter-${type.id}`
              return (
                <div key={type.id} className='flex items-center gap-2'>
                  <Checkbox
                    id={checkboxId}
                    checked={selectedValues.includes(type.id)}
                    onCheckedChange={(checked) => onFilterChange(!!checked, type.id)}
                  />
                  <label htmlFor={checkboxId} className='cursor-pointer text-sm'>
                    {type.title}
                  </label>
                </div>
              )
            })}
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
