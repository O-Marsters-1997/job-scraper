import { createQuery, queryOptions } from "@tanstack/solid-query";
import { loadedData } from "@/lib/queryData";
import { keys } from "../api/keys";
import { fetchSources, resolveUrl } from "../api/sources";

const sourcesQueryOptions = queryOptions({
	queryKey: keys.sources,
	queryFn: fetchSources,
	staleTime: Infinity,
});

export function useSources() {
	return createQuery(() => sourcesQueryOptions);
}

export function useSourceOptions() {
	const sources = useSources();
	return () =>
		(loadedData(sources) ?? []).map((s) => ({ value: s.name, label: s.label }));
}

export function useResolveUrl(
	url: () => string,
	enabled: () => boolean = () => url() !== "",
) {
	return createQuery(() => ({
		queryKey: [...keys.sources, "resolve", url()],
		queryFn: () => resolveUrl(url()),
		enabled: enabled(),
		retry: false,
		staleTime: Infinity,
	}));
}
