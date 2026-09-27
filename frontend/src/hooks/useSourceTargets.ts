import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createSourceTarget,
	deleteSourceTarget,
	fetchSourceTargets,
	rerunSourceTarget,
	updateSourceTarget,
} from "../api/sourceTargets";
import type {
	CreateSourceTargetPayload,
	UpdateSourceTargetPayload,
} from "../types/sourceTarget";

export const sourceTargetsQueryOptions = queryOptions({
	queryKey: keys.sourceTargets,
	queryFn: fetchSourceTargets,
	refetchInterval: 5000,
});

export function useSourceTargets() {
	return createQuery(() => sourceTargetsQueryOptions);
}

export function useRerunSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: rerunSourceTarget,
		onSettled: () =>
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets }),
	}));
}

export function useCreateSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: CreateSourceTargetPayload) =>
			createSourceTarget(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets });
			queryClient.invalidateQueries({ queryKey: keys.companies.all });
		},
	}));
}

export function useUpdateSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({
			id,
			...patch
		}: { id: string } & UpdateSourceTargetPayload) =>
			updateSourceTarget(id, patch),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets });
			queryClient.invalidateQueries({ queryKey: keys.companies.all });
		},
	}));
}

export function useDeleteSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (id: string) => deleteSourceTarget(id),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets });
			queryClient.invalidateQueries({ queryKey: keys.companies.all });
		},
	}));
}
