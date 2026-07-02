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
	type UpdateSourceTargetPayload,
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
		mutationFn: ({
			id,
			...patch
		}: { id: string } & UpdateSourceTargetPayload) =>
			updateSourceTarget(id, patch),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["source-targets"] });
			queryClient.invalidateQueries({ queryKey: ["companies"] });
		},
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
