import { createConnectQueryKey, useMutation, useSuspenseQuery } from '@connectrpc/connect-query'
import { assignPlatformRole, getUser, listUsers, preRegisterUser, sendTelegramNotification } from '@studyverse/proto'
import { useQueryClient } from '@tanstack/react-query'

export function usersListKey() {
  return createConnectQueryKey({ schema: listUsers, cardinality: 'finite' })
}

export function userDetailKey(userId: number) {
  return createConnectQueryKey({ schema: getUser, input: { id: userId }, cardinality: 'finite' })
}

export function useUsers(input?: { limit?: number; page?: number }) {
  return useSuspenseQuery(listUsers, input ?? {})
}

export function useUser(userId: number) {
  return useSuspenseQuery(getUser, { id: userId })
}

export function useAssignPlatformRole() {
  const queryClient = useQueryClient()
  return useMutation(assignPlatformRole, {
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: usersListKey() })
      if (variables.userId) {
        queryClient.invalidateQueries({ queryKey: userDetailKey(variables.userId) })
      }
    }
  })
}

export function usePreRegisterUser() {
  const queryClient = useQueryClient()
  return useMutation(preRegisterUser, {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: usersListKey() })
    }
  })
}

export function useSendTelegramNotification() {
  return useMutation(sendTelegramNotification)
}
