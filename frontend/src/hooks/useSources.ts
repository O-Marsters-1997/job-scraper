import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchSources, resolveUrl } from "../api/sources";

export const sourcesQueryOptions = queryOptions({
	queryKey: keys.sources,
	queryFn: fetchSources,
	staleTime: Infinity,
});

export function useSources() {
	return createQuery(() => sourcesQueryOptions);
}

export function useResolveUrl(url: () => string) {
	return createQuery(() => ({
		queryKey: [...keys.sources, "resolve", url()],
		queryFn: () => resolveUrl(url()),
		enabled: url() !== "",
		retry: false,
		staleTime: Infinity,
	}));
}
