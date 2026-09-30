import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringConfig, updateScoringConfig } from "../api/scoringConfig";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const scoringConfigQueryOptions = queryOptions({
	queryKey: keys.scoringConfig,
	queryFn: fetchScoringConfig,
});

export function useScoringConfig() {
	return createQuery(() => scoringConfigQueryOptions);
}

export function useUpdateScoringConfig() {
	return useInvalidatingMutation(updateScoringConfig, [
		keys.scoringConfig,
		keys.scores.status(),
	]);
}
