import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchScoringOptions } from "../api/scoringOptions";

export const scoringOptionsQueryOptions = queryOptions({
	queryKey: ["scoring-options"],
	queryFn: fetchScoringOptions,
});

export function useScoringOptions() {
	return createQuery(() => scoringOptionsQueryOptions);
}
