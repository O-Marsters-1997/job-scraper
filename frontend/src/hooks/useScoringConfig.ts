import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringConfig, updateScoringConfig } from "../api/scoringConfig";
import type { ScoringConfigInput } from "../types/scoringConfig";

export const scoringConfigQueryOptions = queryOptions({
	queryKey: keys.scoringConfig,
	queryFn: fetchScoringConfig,
});

export function useScoringConfig() {
	return createQuery(() => scoringConfigQueryOptions);
}

export function useUpdateScoringConfig() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: ScoringConfigInput) => updateScoringConfig(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.scoringConfig });
			queryClient.invalidateQueries({ queryKey: keys.scores.status() });
		},
	}));
}
