import { type ScrapeRunPage, scrapeRunPageSchema } from "../types/scrapeRun";
import { apiFetch } from "./client";
import { mocked } from "./config";

export interface ScrapeRunsParams {
	target?: string | undefined;
	before?: string | undefined;
}

export async function fetchScrapeRuns(
	params: ScrapeRunsParams,
): Promise<ScrapeRunPage> {
	return mocked(
		(db) => db.getScrapeRuns(params),
		() => {
			const query = new URLSearchParams();
			if (params.target) query.set("target", params.target);
			if (params.before) query.set("before", params.before);
			return apiFetch(`/scrape-runs?${query}`, scrapeRunPageSchema);
		},
	);
}
