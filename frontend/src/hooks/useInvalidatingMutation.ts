import {
	createMutation,
	type QueryKey,
	useQueryClient,
} from "@tanstack/solid-query";

type Invalidate<TData> =
	| readonly QueryKey[]
	| ((result: TData) => readonly QueryKey[]);

export function useInvalidatingMutation<TData, TVariables = void>(
	mutationFn: (variables: TVariables) => Promise<TData>,
	invalidate: Invalidate<TData>,
	opts?: { onSettled?: boolean },
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
		...(opts?.onSettled
			? { onSettled: invalidateFor }
			: { onSuccess: invalidateFor }),
	}));
}
