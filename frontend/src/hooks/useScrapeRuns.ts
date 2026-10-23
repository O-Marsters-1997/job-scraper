import { createQuery, keepPreviousData } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScrapeRuns, type ScrapeRunsParams } from "../api/scrapeRuns";

const LIVE_POLL_MS = 5000;

export const scrapeRunsQuery = (params: ScrapeRunsParams) => ({
	queryKey: keys.scrapeRuns(params),
	queryFn: () => fetchScrapeRuns(params),
});

export function useScrapeRuns(params: () => ScrapeRunsParams) {
	return createQuery(() => ({
		...scrapeRunsQuery(params()),
		placeholderData: keepPreviousData,
		refetchInterval: params().before ? false : LIVE_POLL_MS,
	}));
}
