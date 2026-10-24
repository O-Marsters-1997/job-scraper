import {
	createInfiniteQuery,
	createQuery,
	keepPreviousData,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	fetchScrapeRunSummary,
	fetchScrapeRuns,
	fetchSourceHealth,
	type ScrapeRunFilters,
	type ScrapeRunsParams,
} from "../api/scrapeRuns";

const LIVE_POLL_MS = 5000;

export const scrapeRunsQuery = (params: ScrapeRunsParams) => ({
	queryKey: keys.scrapeRuns(params),
	queryFn: () => fetchScrapeRuns(params),
});

export const scrapeRunSummaryQuery = (params: ScrapeRunFilters) => ({
	queryKey: keys.scrapeRunSummary(params),
	queryFn: () => fetchScrapeRunSummary(params),
});

export function useScrapeRuns(params: () => ScrapeRunsParams) {
	return createQuery(() => ({
		...scrapeRunsQuery(params()),
		placeholderData: keepPreviousData,
		refetchInterval: params().before ? false : LIVE_POLL_MS,
	}));
}

export function useScrapeRunPages(params: () => ScrapeRunFilters) {
	return createInfiniteQuery(() => ({
		queryKey: [...keys.scrapeRuns(params()), "pages"],
		initialPageParam: undefined as string | undefined,
		queryFn: ({ pageParam }) =>
			fetchScrapeRuns({ ...params(), before: pageParam }),
		getNextPageParam: (last) => last.NextCursor ?? undefined,
	}));
}

export function useSourceHealth() {
	return createQuery(() => ({
		queryKey: keys.sourceHealth,
		queryFn: fetchSourceHealth,
		refetchInterval: LIVE_POLL_MS,
	}));
}

export function useScrapeRunSummary(
	params: () => ScrapeRunFilters,
	live: () => boolean,
) {
	return createQuery(() => ({
		...scrapeRunSummaryQuery(params()),
		placeholderData: keepPreviousData,
		refetchInterval: live() ? LIVE_POLL_MS : false,
	}));
}
