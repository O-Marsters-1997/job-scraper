import assert from "node:assert/strict";
import { setDemoData } from "../lib/demoData";
import { applyJobFilters, DEFAULT_FILTERS } from "../lib/jobFilters";
import { API_BASE } from "./config";
import { fetchAllJobs, fetchJob } from "./jobs";

setDemoData(true);
const [first] = await fetchAllJobs();
const detail = await fetchJob(first?.ID ?? "");
assert.ok(detail.Description, "job detail must include full description");
setDemoData(false);

const originalFetch = globalThis.fetch;
const requested: string[] = [];
globalThis.fetch = async (input) => {
	requested.push(String(input));
	const page = Number(new URL(String(input)).searchParams.get("page"));
	const ids = page === 1 ? [0, 1] : [2];
	return Response.json({
		total: 3,
		items: ids.map((i) => ({
			ID: String(i),
			Title: i === 2 ? "Engineer" : "Designer",
			CompanySlug: "acme",
			Location: "London",
			URL: "https://example.com",
			Source: "test",
			PostedAt: "2026-01-01T00:00:00Z",
			ScrapedAt: "2026-01-01T00:00:00Z",
			SuitabilityScore: null,
		})),
	});
};
try {
	const all = await fetchAllJobs();
	const filtered = applyJobFilters(all, { ...DEFAULT_FILTERS, q: "Engineer" });
	assert.deepEqual(requested, [
		`${API_BASE}/jobs?scored=1&page=1&page_size=100`,
		`${API_BASE}/jobs?scored=1&page=2&page_size=100`,
	]);
	assert.equal(
		filtered[0]?.ID,
		"2",
		"job filters must see jobs beyond the first server page",
	);
} finally {
	globalThis.fetch = originalFetch;
}
