import type { Category } from '@studyverse/proto'
import { createFileRoute } from '@tanstack/react-router'
import type { RowSelectionState } from '@tanstack/react-table'
import { Loader2Icon, PlusIcon, Trash2Icon } from 'lucide-react'
import { Suspense, useCallback, useState } from 'react'
import { toast } from 'sonner'
import {
  CategoryModal,
  CategorySuggestions,
  createColumns,
  useCategories,
  useDeleteCategory
} from '@/features/categories'
import { DataTable, useDataTableState } from '@/shared/components/admin-data-table'
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
import { Skeleton } from '@/shared/ui/components/skeleton'

export const Route = createFileRoute('/_authenticated/categories')({
  component: () => (
    <Suspense fallback={<CategoriesLoading />}>
      <Categories />
    </Suspense>
  )
})

const loadingRowIds = ['row-1', 'row-2', 'row-3', 'row-4', 'row-5', 'row-6', 'row-7', 'row-8', 'row-9', 'row-10']

function CategoriesLoading() {
  return (
    <div className='space-y-4'>
      <div className='mb-6 space-y-2'>
        <Skeleton className='h-8 w-32' />
        <Skeleton className='h-5 w-64' />
      </div>

      <div className='mb-4 flex items-center justify-between gap-3'>
        <Skeleton className='h-10 w-64' />
        <Skeleton className='h-10 w-32' />
      </div>

      <div className='rounded-lg border'>
        <div className='border-b px-4 py-3'>
          <Skeleton className='h-4 w-full max-w-xl' />
        </div>
        {loadingRowIds.map((rowId) => (
          <div key={rowId} className='border-b px-4 py-3 last:border-0'>
            <Skeleton className='h-5 w-full' />
          </div>
        ))}
      </div>

      <div className='mt-4 flex items-center justify-between'>
        <Skeleton className='h-9 w-28' />
        <Skeleton className='h-5 w-32' />
        <Skeleton className='h-9 w-24' />
      </div>
    </div>
  )
}

function Categories() {
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingCategory, setEditingCategory] = useState<Category | undefined>(undefined)
  const { tableState } = useDataTableState({ defaultSortBy: 'name', defaultSortDesc: false })
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const [isBulkDeleteDialogOpen, setIsBulkDeleteDialogOpen] = useState(false)
  const [pendingBulkDeleteIds, setPendingBulkDeleteIds] = useState<number[]>([])

  const deleteMutation = useDeleteCategory()

  const hasActiveFilters = Boolean(tableState.search)

  const { data } = useCategories({
    limit: hasActiveFilters ? 1000 : tableState.pageSize,
    page: hasActiveFilters ? 1 : tableState.page
  })

  const handleAddCategory = () => {
    setEditingCategory(undefined)
    setIsModalOpen(true)
  }

  const handleEditCategory = (category: Category) => {
    setEditingCategory(category)
    setIsModalOpen(true)
  }

  const handleDeleteSelected = useCallback((ids: (string | number)[]) => {
    setPendingBulkDeleteIds(ids.map((id) => Number(id)))
    setIsBulkDeleteDialogOpen(true)
  }, [])

  const handleRowSelectionChange = useCallback(
    (updater: RowSelectionState | ((old: RowSelectionState) => RowSelectionState)) => {
      setRowSelection(typeof updater === 'function' ? updater(rowSelection) : updater)
    },
    [rowSelection]
  )

  const handleClearSelection = useCallback(() => {
    setRowSelection({})
  }, [])

  const handleConfirmDelete = useCallback(async () => {
    try {
      for (const id of pendingBulkDeleteIds) {
        await deleteMutation.mutateAsync({ id })
      }
      toast.success(
        pendingBulkDeleteIds.length === 1 ? 'Category deleted' : `${pendingBulkDeleteIds.length} categories deleted`
      )
      setRowSelection({})
      setIsBulkDeleteDialogOpen(false)
      setPendingBulkDeleteIds([])
    } catch (error) {
      toast.error('Failed to delete category')
      console.error(error)
    }
  }, [pendingBulkDeleteIds, deleteMutation])

  const selectedCategoryIds = Object.keys(rowSelection).map((id) => parseInt(id, 10))
  const selectedCount = selectedCategoryIds.length

  const columns = createColumns({ onEdit: handleEditCategory })

  return (
    <div className='space-y-6'>
      <div>
        <h1 className='text-2xl font-semibold tracking-tight'>Categories</h1>
        <p className='text-sm text-muted-foreground mt-1'>Manage event categories and their colors</p>
      </div>

      <CategorySuggestions />

      <DataTable
        data={data.categories}
        columns={columns}
        totalCount={data.total || 0}
        hasActiveFilters={hasActiveFilters}
        searchColumnId='name'
        searchPlaceholder='Search categories...'
        entityName='categories'
        defaultSortBy='name'
        defaultSortDesc={false}
        rowSelection={rowSelection}
        onRowClick={handleEditCategory}
        onRowSelectionChange={handleRowSelectionChange}
        onDeleteSelected={handleDeleteSelected}
        getRowId={(row) => String(row.id)}
        isDeleting={deleteMutation.isPending}
        toolbarActions={
          <Button className='ml-auto' onClick={handleAddCategory}>
            <PlusIcon aria-hidden='true' className='-ms-1 opacity-60' size={16} />
            Add category
          </Button>
        }
      />

      <FloatingActionBar
        selectedCount={selectedCount}
        totalCount={data.total || 0}
        onClearSelection={handleClearSelection}
      >
        <Button
          variant='destructive'
          size='sm'
          onClick={() => handleDeleteSelected(selectedCategoryIds)}
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
              Delete {pendingBulkDeleteIds.length === 1 ? 'Category' : `${pendingBulkDeleteIds.length} Categories`}
            </AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete{' '}
              {pendingBulkDeleteIds.length === 1 ? 'this category' : `these ${pendingBulkDeleteIds.length} categories`}.
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

      <CategoryModal open={isModalOpen} onOpenChange={setIsModalOpen} category={editingCategory} />
    </div>
  )
}
