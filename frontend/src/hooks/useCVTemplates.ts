import { createMutation, createQuery, queryOptions, useQueryClient } from '@tanstack/solid-query'
import { addTrackedDoc, fetchCVTemplates, removeTrackedDoc } from '../api/cvTemplates'

export const cvTemplatesQueryOptions = queryOptions({
	queryKey: ['cv-templates'],
	queryFn: fetchCVTemplates,
})

export function useCVTemplates() {
	return createQuery(() => cvTemplatesQueryOptions)
}

export function useAddTrackedDoc() {
	const queryClient = useQueryClient()
	return createMutation(() => ({
		mutationFn: (url: string) => addTrackedDoc(url),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ['cv-templates'] }),
	}))
}

export function useRemoveTrackedDoc() {
	const queryClient = useQueryClient()
	return createMutation(() => ({
		mutationFn: (docId: string) => removeTrackedDoc(docId),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ['cv-templates'] }),
	}))
}
