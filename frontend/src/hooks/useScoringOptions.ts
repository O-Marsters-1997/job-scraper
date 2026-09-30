import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringOptions } from "../api/scoringOptions";

const scoringOptionsQueryOptions = queryOptions({
	queryKey: keys.scoringOptions,
	queryFn: fetchScoringOptions,
});

export function useScoringOptions() {
	return createQuery(() => scoringOptionsQueryOptions);
}
