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
let requested = "";
globalThis.fetch = async (input) => {
	requested = String(input);
	return Response.json(
		Array.from({ length: 11 }, (_, i) => ({
			ID: String(i),
			Title: i === 10 ? "Engineer" : "Designer",
			CompanySlug: "acme",
			Location: "London",
			URL: "https://example.com",
			Source: "test",
			UpdatedAt: "2026-01-01T00:00:00Z",
			ScrapedAt: "2026-01-01T00:00:00Z",
			SuitabilityScore: null,
		})),
	);
};
try {
	const all = await fetchAllJobs();
	const filtered = applyJobFilters(all, { ...DEFAULT_FILTERS, q: "Engineer" });
	assert.equal(requested, `${API_BASE}/jobs/all`);
	assert.equal(
		filtered[0]?.ID,
		"10",
		"job filters must see jobs beyond the first server page",
	);
} finally {
	globalThis.fetch = originalFetch;
}
