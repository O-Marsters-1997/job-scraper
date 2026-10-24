import {
	type ScrapeRunPage,
	type ScrapeRunSummary,
	type SourceHealth,
	scrapeRunPageSchema,
	scrapeRunSummarySchema,
	sourceHealthListSchema,
} from "../types/scrapeRun";
import { apiFetch } from "./client";
import { mocked } from "./config";

export interface ScrapeRunFilters {
	target?: string | undefined;
	q?: string | undefined;
	outcome?: string | undefined;
	kind?: string | undefined;
	source?: string | undefined;
	trigger?: string | undefined;
	range?: string | undefined;
	from?: string | undefined;
	to?: string | undefined;
}

export interface ScrapeRunsParams extends ScrapeRunFilters {
	before?: string | undefined;
}

function queryString(params: ScrapeRunsParams): string {
	const query = new URLSearchParams();
	for (const [key, value] of Object.entries(params)) {
		if (value) query.set(key, value);
	}
	return query.toString();
}

export async function fetchScrapeRuns(
	params: ScrapeRunsParams,
): Promise<ScrapeRunPage> {
	return mocked(
		(db) => db.getScrapeRuns(params),
		() => apiFetch(`/scrape-runs?${queryString(params)}`, scrapeRunPageSchema),
	);
}

export async function fetchScrapeRunSummary(
	params: ScrapeRunFilters,
): Promise<ScrapeRunSummary> {
	return mocked(
		(db) => db.getScrapeRunSummary(params),
		() =>
			apiFetch(
				`/scrape-runs/summary?${queryString(params)}`,
				scrapeRunSummarySchema,
			),
	);
}

export async function fetchSourceHealth(): Promise<SourceHealth[]> {
	return mocked(
		(db) => db.getSourceHealth(),
		() => apiFetch("/scrape-runs/health", sourceHealthListSchema),
	);
}
