import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringConfig, updateScoringConfig } from "../api/scoringConfig";
import type { ScoringConfig } from "../types/scoringConfig";

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
		mutationFn: (payload: ScoringConfig) => updateScoringConfig(payload),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: keys.scoringConfig }),
	}));
}
