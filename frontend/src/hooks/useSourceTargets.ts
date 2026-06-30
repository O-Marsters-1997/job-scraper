import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	type CreateSourceTargetPayload,
	createSourceTarget,
	deleteSourceTarget,
	fetchSourceTargets,
	updateSourceTarget,
} from "../api/sourceTargets";

export const sourceTargetsQueryOptions = queryOptions({
	queryKey: ["source-targets"],
	queryFn: fetchSourceTargets,
});

export function useSourceTargets() {
	return createQuery(() => sourceTargetsQueryOptions);
}

export function useCreateSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: CreateSourceTargetPayload) =>
			createSourceTarget(payload),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["source-targets"] }),
	}));
}

export function useUpdateSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
			updateSourceTarget(id, enabled),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["source-targets"] }),
	}));
}

export function useDeleteSourceTarget() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (id: string) => deleteSourceTarget(id),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: ["source-targets"] }),
	}));
}
