import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchSources } from "../api/sources";

export const sourcesQueryOptions = queryOptions({
	queryKey: keys.sources,
	queryFn: fetchSources,
	staleTime: Infinity,
});

export function useSources() {
	return createQuery(() => sourcesQueryOptions);
}
