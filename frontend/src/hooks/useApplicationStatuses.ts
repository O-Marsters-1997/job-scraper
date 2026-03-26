import { createQuery, createMutation, useQueryClient, queryOptions } from '@tanstack/solid-query'
import {
  fetchApplicationStatuses,
  createApplicationStatus,
  updateApplicationStatus,
  deleteApplicationStatus,
} from '../api/applicationStatuses'

export const applicationStatusesQueryOptions = queryOptions({
  queryKey: ['application-statuses'],
  queryFn: fetchApplicationStatuses,
})

export function useApplicationStatuses() {
  return createQuery(() => applicationStatusesQueryOptions)
}

export function useCreateApplicationStatus() {
  const queryClient = useQueryClient()
  return createMutation(() => ({
    mutationFn: ({ name, colour }: { name: string; colour: string }) =>
      createApplicationStatus(name, colour),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['application-statuses'] }),
  }))
}

export function useUpdateApplicationStatus() {
  const queryClient = useQueryClient()
  return createMutation(() => ({
    mutationFn: ({ id, name, colour }: { id: string; name: string; colour: string }) =>
      updateApplicationStatus(id, name, colour),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['application-statuses'] }),
  }))
}

export function useDeleteApplicationStatus() {
  const queryClient = useQueryClient()
  return createMutation(() => ({
    mutationFn: (id: string) => deleteApplicationStatus(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['application-statuses'] }),
  }))
}
