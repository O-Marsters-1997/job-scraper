import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringConfig, updateScoringConfig } from "../api/scoringConfig";
import type { ScoringConfigInput } from "../types/scoringConfig";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const scoringConfigQueryOptions = queryOptions({
	queryKey: keys.scoringConfig,
	queryFn: fetchScoringConfig,
});

export function useScoringConfig() {
	return createQuery(() => scoringConfigQueryOptions);
}

export function useUpdateScoringConfig() {
	return useInvalidatingMutation(
		(payload: ScoringConfigInput) => updateScoringConfig(payload),
		[keys.scoringConfig, keys.scores.status()],
	);
}
