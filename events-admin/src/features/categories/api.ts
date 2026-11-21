import { createConnectQueryKey, useMutation, useQuery, useSuspenseQuery } from '@connectrpc/connect-query'
import { createCategory, deleteCategory, listCategories, updateCategory } from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'

export function categoriesListKey() {
  return createConnectQueryKey({ schema: listCategories, cardinality: 'finite' })
}

export function useCategories(input?: { limit?: number; page?: number }) {
  return useSuspenseQuery(listCategories, input ?? {})
}

export function useCategoryCatalog() {
  return useQuery(listCategories, { page: 1, limit: 1000 })
}

export function useCreateCategory() {
  const queryClient = useQueryClient()
  return useMutation(createCategory, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoriesListKey() })
    }
  })
}

export function useUpdateCategory() {
  const queryClient = useQueryClient()
  return useMutation(updateCategory, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoriesListKey() })
    }
  })
}

export function useDeleteCategory() {
  const queryClient = useQueryClient()
  return useMutation(deleteCategory, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: categoriesListKey() })
    }
  })
}
