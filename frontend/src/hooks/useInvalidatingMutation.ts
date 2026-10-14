import {
	createMutation,
	type QueryClient,
	type QueryKey,
	useQueryClient,
} from "@tanstack/solid-query";

type Invalidate<TData> =
	| readonly QueryKey[]
	| ((result: TData) => readonly QueryKey[]);

export function useInvalidatingMutation<TData, TVariables = void>(
	mutationFn: (variables: TVariables) => Promise<TData>,
	invalidate: Invalidate<TData>,
	opts?: {
		onSettled?: boolean;
		patch?: (
			queryClient: QueryClient,
			result: TData,
			variables: TVariables,
		) => void;
	},
) {
	const queryClient = useQueryClient();
	const invalidateFor = (result: TData | undefined) => {
		const keys =
			typeof invalidate === "function"
				? result !== undefined
					? invalidate(result)
					: []
				: invalidate;
		for (const queryKey of keys) queryClient.invalidateQueries({ queryKey });
	};
	return createMutation(() => ({
		mutationFn,
		onSuccess: (result: TData, variables: TVariables) => {
			opts?.patch?.(queryClient, result, variables);
			if (!opts?.onSettled) invalidateFor(result);
		},
		...(opts?.onSettled ? { onSettled: invalidateFor } : {}),
	}));
}
