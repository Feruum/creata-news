import type { Event } from '@studyverse/proto'
import { createFileRoute } from '@tanstack/react-router'
import type { RowSelectionState } from '@tanstack/react-table'
import { DownloadIcon, Loader2Icon, PlusIcon, Trash2Icon, UploadIcon } from 'lucide-react'
import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryState, useQueryStates } from 'nuqs'
import { Suspense, useCallback, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { buildCategoryOptions, useCategoryCatalog } from '@/features/categories'
import {
  CreateEventForm,
  columns,
  EventDetailsModalControlled,
  EventsFilter,
  EventsLoading,
  ExcelImportDialog,
  MultiStepEventForm,
  SimpleEventCardList,
  useDeleteEvent,
  useEventsForAdmin
} from '@/features/events'
import { formatOptions, getEventStatus, statusOptions } from '@/features/events/utils/event-status'
import { DataTable } from '@/shared/components/admin-data-table'
import { exportEventsToExcel, exportSelectedEventsToExcel } from '@/shared/lib/excel-export'
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
import { Button, buttonVariants } from '@/shared/ui/components/button'
import { FloatingActionBar } from '@/shared/ui/components/floating-action-bar'

export const Route = createFileRoute('/_authenticated/events')({
  component: () => (
    <Suspense fallback={<EventsLoading />}>
      <Events />
    </Suspense>
  )
})

function Events() {
  const [isImportDialogOpen, setIsImportDialogOpen] = useState(false)
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isMultiStepFormOpen, setIsMultiStepFormOpen] = useState(false)
  const [selectedEventId, setSelectedEventId] = useQueryState('event', parseAsInteger)
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})

  const [isBulkDeleteDialogOpen, setIsBulkDeleteDialogOpen] = useState(false)
  const [pendingBulkDeleteIds, setPendingBulkDeleteIds] = useState<number[]>([])

  const deleteMutation = useDeleteEvent()

  const [formatFilter, setFormatFilter] = useQueryState('format', parseAsArrayOf(parseAsInteger).withDefault([]))

  const [statusFilter, setStatusFilter] = useQueryState('status', parseAsArrayOf(parseAsString).withDefault([]))

  const [orgFilter, setOrgFilter] = useQueryState('org', parseAsArrayOf(parseAsInteger).withDefault([]))

  const [categoryFilter, setCategoryFilter] = useQueryState('category', parseAsArrayOf(parseAsInteger).withDefault([]))

  const [actionState, setActionState] = useQueryStates({
    action: parseAsString.withDefault('')
  })

  const isCreateFromUrl = actionState.action === 'create'
  const isImportFromUrl = actionState.action === 'import'

  const handleCreateDialogChange = useCallback(
    (open: boolean) => {
      setIsCreateDialogOpen(open)
      if (!open && isCreateFromUrl) {
        setActionState({ action: null })
      }
    },
    [isCreateFromUrl, setActionState]
  )

  const handleImportDialogChange = useCallback(
    (open: boolean) => {
      setIsImportDialogOpen(open)
      if (!open && isImportFromUrl) {
        setActionState({ action: null })
      }
    },
    [isImportFromUrl, setActionState]
  )

  const activeFilterCount = formatFilter.length + statusFilter.length + orgFilter.length + categoryFilter.length

  const { data } = useEventsForAdmin({
    limit: 1000,
    page: 1
  })
  const categoryCatalog = useCategoryCatalog()

  const organizationOptions = useMemo(() => {
    const orgMap = new Map<number, string>()
    data.events.forEach((event: Event) => {
      if (event.organization?.id && event.organization?.title) {
        orgMap.set(event.organization.id, event.organization.title)
      }
    })
    return Array.from(orgMap.entries())
      .map(([id, title]) => ({ value: id, label: title }))
      .sort((a, b) => a.label.localeCompare(b.label))
  }, [data.events])

  const categoryOptions = useMemo(() => {
    return buildCategoryOptions(
      categoryCatalog.data?.categories ?? [],
      data.events.flatMap((event) => (event.category ? [event.category] : []))
    )
  }, [data.events, categoryCatalog.data])

  const filteredEvents = useMemo(() => {
    let events = data.events

    if (formatFilter.length > 0) {
      events = events.filter((event: Event) => formatFilter.includes(event.format))
    }

    if (statusFilter.length > 0) {
      events = events.filter((event: Event) => statusFilter.includes(getEventStatus(event)))
    }

    if (orgFilter.length > 0) {
      events = events.filter((event: Event) => event.organization?.id && orgFilter.includes(event.organization.id))
    }

    if (categoryFilter.length > 0) {
      events = events.filter((event: Event) => event.category?.id && categoryFilter.includes(event.category.id))
    }

    return events
  }, [data.events, formatFilter, statusFilter, orgFilter, categoryFilter])

  const handleFormatChange = (checked: boolean, value: number) => {
    const newValues = formatFilter ? [...formatFilter] : []
    if (checked) {
      newValues.push(value)
    } else {
      const index = newValues.indexOf(value)
      if (index > -1) {
        newValues.splice(index, 1)
      }
    }
    setFormatFilter(newValues.length ? newValues : null)
  }

  const handleStatusChange = (checked: boolean, value: string) => {
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

  const handleOrgChange = (checked: boolean, value: number) => {
    const newValues = orgFilter ? [...orgFilter] : []
    if (checked) {
      newValues.push(value)
    } else {
      const index = newValues.indexOf(value)
      if (index > -1) {
        newValues.splice(index, 1)
      }
    }
    setOrgFilter(newValues.length ? newValues : null)
  }

  const handleCategoryChange = (checked: boolean, value: number) => {
    const newValues = categoryFilter ? [...categoryFilter] : []
    if (checked) {
      newValues.push(value)
    } else {
      const index = newValues.indexOf(value)
      if (index > -1) {
        newValues.splice(index, 1)
      }
    }
    setCategoryFilter(newValues.length ? newValues : null)
  }

  const handleRowClick = useCallback(
    (event: Event) => {
      void setSelectedEventId(event.id)
    },
    [setSelectedEventId]
  )

  const handleRowSelectionChange = useCallback((selection: RowSelectionState) => {
    setRowSelection(selection)
  }, [])

  const handleClearSelection = useCallback(() => {
    setRowSelection({})
  }, [])

  const selectedEventIds = Object.keys(rowSelection).map((id) => parseInt(id, 10))
  const selectedCount = selectedEventIds.length

  const handleDeleteSelected = useCallback((ids: (string | number)[]) => {
    setPendingBulkDeleteIds(ids.map((id) => Number(id)))
    setIsBulkDeleteDialogOpen(true)
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    try {
      for (const id of pendingBulkDeleteIds) {
        await deleteMutation.mutateAsync({ id })
      }
      toast.success(
        pendingBulkDeleteIds.length === 1 ? 'Event deleted' : `${pendingBulkDeleteIds.length} events deleted`
      )
      setRowSelection({})
      setIsBulkDeleteDialogOpen(false)
      setPendingBulkDeleteIds([])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to delete event')
    }
  }, [pendingBulkDeleteIds, deleteMutation])

  const handleExportAll = useCallback(async () => {
    try {
      await exportEventsToExcel(filteredEvents)
      toast.success('Export completed', {
        description: `Exported ${filteredEvents.length} events to Excel`
      })
    } catch (error) {
      toast.error('Export failed', {
        description: error instanceof Error ? error.message : 'Unknown error'
      })
    }
  }, [filteredEvents])

  const handleExportSelected = useCallback(async () => {
    try {
      await exportSelectedEventsToExcel(filteredEvents, selectedEventIds)
      toast.success('Export completed', {
        description: `Exported ${selectedEventIds.length} selected events to Excel`
      })
    } catch (error) {
      toast.error('Export failed', {
        description: error instanceof Error ? error.message : 'Unknown error'
      })
    }
  }, [filteredEvents, selectedEventIds])

  return (
    <div className='space-y-6'>
      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Events</h1>
        <p className='text-sm text-muted-foreground'>Manage events and activities</p>
      </div>

      <div className='hidden md:block'>
        <DataTable
          data={filteredEvents}
          columns={columns}
          totalCount={filteredEvents.length}
          hasActiveFilters={true}
          searchColumnId='title'
          searchPlaceholder='Search events...'
          entityName='events'
          defaultSortBy='startTime'
          defaultSortDesc={true}
          enableVirtualization={false}
          enableDensityToggle={true}
          enableMultiSort={true}
          rowSelection={rowSelection}
          onRowClick={handleRowClick}
          onRowSelectionChange={handleRowSelectionChange}
          onDeleteSelected={handleDeleteSelected}
          getRowId={(row) => String(row.id)}
          filterComponents={
            <EventsFilter
              formatFilter={formatFilter}
              statusFilter={statusFilter}
              orgFilter={orgFilter}
              categoryFilter={categoryFilter}
              organizationOptions={organizationOptions}
              categoryOptions={categoryOptions}
              activeFilterCount={activeFilterCount}
              statusOptions={statusOptions}
              formatOptions={formatOptions}
              onFormatChange={handleFormatChange}
              onStatusChange={handleStatusChange}
              onOrgChange={handleOrgChange}
              onCategoryChange={handleCategoryChange}
            />
          }
          toolbarActions={
            <>
              <Button variant='outline' size='sm' onClick={handleExportAll} className='h-9'>
                <DownloadIcon className='h-4 w-4 mr-2' />
                Export
              </Button>
              <Button variant='outline' size='sm' onClick={() => setIsImportDialogOpen(true)} className='h-9'>
                <UploadIcon className='h-4 w-4 mr-2' />
                Import
              </Button>
              {import.meta.env.DEV && (
                <Button
                  variant='ghost'
                  size='sm'
                  onClick={() => setIsMultiStepFormOpen(true)}
                  className='h-9 text-orange-600 hover:text-orange-700 hover:bg-orange-50'
                >
                  Test Multi-Step
                </Button>
              )}
              <Button size='sm' onClick={() => setIsCreateDialogOpen(true)} className='h-9'>
                <PlusIcon className='h-4 w-4 mr-2' />
                Add Event
              </Button>
            </>
          }
        />
      </div>

      <div className='md:hidden'>
        <div className='space-y-4'>
          <div className='flex gap-2'>
            <Button variant='outline' onClick={() => setIsImportDialogOpen(true)} size='icon'>
              <UploadIcon size={16} />
            </Button>
            <Button onClick={() => setIsCreateDialogOpen(true)} className='flex-1'>
              <PlusIcon className='mr-2' size={16} />
              Add event
            </Button>
          </div>
          <SimpleEventCardList events={filteredEvents} />
        </div>
      </div>

      <FloatingActionBar
        selectedCount={selectedCount}
        totalCount={data.total || 0}
        onClearSelection={handleClearSelection}
      >
        <Button variant='outline' size='sm' onClick={handleExportSelected} className='gap-2'>
          <DownloadIcon className='h-4 w-4' />
          Export
        </Button>
        <Button
          variant='destructive'
          size='sm'
          onClick={() => handleDeleteSelected(selectedEventIds)}
          className='gap-2'
        >
          <Trash2Icon className='h-4 w-4' />
          Delete
        </Button>
      </FloatingActionBar>

      <AlertDialog open={isBulkDeleteDialogOpen} onOpenChange={setIsBulkDeleteDialogOpen}>
        <AlertDialogContent style={{ overscrollBehavior: 'contain' }}>
          <AlertDialogHeader>
            <AlertDialogTitle>
              Delete {pendingBulkDeleteIds.length === 1 ? 'Event' : `${pendingBulkDeleteIds.length} Events`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete{' '}
              {pendingBulkDeleteIds.length === 1 ? 'this event' : `these ${pendingBulkDeleteIds.length} events`}.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
              className={buttonVariants({ variant: 'destructive' })}
            >
              {deleteMutation.isPending && <Loader2Icon className='mr-2 h-4 w-4 animate-spin' aria-hidden='true' />}
              {deleteMutation.isPending ? 'Deleting…' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <ExcelImportDialog open={isImportDialogOpen || isImportFromUrl} onOpenChange={handleImportDialogChange} />
      <CreateEventForm
        open={isCreateDialogOpen || isCreateFromUrl}
        onOpenChange={handleCreateDialogChange}
        onCreatedDraft={(eventId) => {
          void setSelectedEventId(eventId)
        }}
      />
      <MultiStepEventForm open={isMultiStepFormOpen} onOpenChange={setIsMultiStepFormOpen} />
      <EventDetailsModalControlled eventId={selectedEventId} onClose={() => setSelectedEventId(null)} />
    </div>
  )
}
