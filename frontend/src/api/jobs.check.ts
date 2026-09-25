import { setDemoData } from "../lib/demoData";
import { applyJobFilters, DEFAULT_FILTERS } from "../lib/jobFilters";
import { API_BASE } from "./config";
import { fetchAllJobs, fetchJob, fetchJobs } from "./jobs";

setDemoData(true);
const first = await fetchJobs({ limit: 2 });
const second = await fetchJobs({ limit: 2, cursor: first.next_cursor });
if (
	first.items.length !== 2 ||
	!first.next_cursor ||
	second.items[0]?.ID === first.items[0]?.ID
) {
	throw new Error("job pages must be bounded and advance");
}
const detail = await fetchJob(first.items[0]?.ID ?? "");
if (!detail.Description)
	throw new Error("job detail must include full description");
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
			Source: "test",
		})),
	);
};
try {
	const all = await fetchAllJobs();
	const filtered = applyJobFilters(all, { ...DEFAULT_FILTERS, q: "Engineer" });
	if (requested !== `${API_BASE}/jobs/all` || filtered[0]?.ID !== "10")
		throw new Error("job filters must see jobs beyond the first server page");
} finally {
	globalThis.fetch = originalFetch;
}
