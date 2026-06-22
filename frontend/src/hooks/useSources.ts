import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchSources } from "../api/sources";

export const sourcesQueryOptions = queryOptions({
	queryKey: ["sources"],
	queryFn: fetchSources,
	staleTime: Infinity,
});

export function useSources() {
	return createQuery(() => sourcesQueryOptions);
}
