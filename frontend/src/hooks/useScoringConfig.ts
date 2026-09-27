import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	fetchScoringConfig,
	type ScoringConfig,
	updateScoringConfig,
} from "../api/scoringConfig";

export const scoringConfigQueryOptions = queryOptions({
	queryKey: ["scoring-config"],
	queryFn: fetchScoringConfig,
});

export function useScoringConfig() {
	return createQuery(() => scoringConfigQueryOptions);
}

export function useUpdateScoringConfig() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: ScoringConfig) => updateScoringConfig(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["scoring-config"] });
			queryClient.invalidateQueries({ queryKey: ["jobs"] });
		},
	}));
}
