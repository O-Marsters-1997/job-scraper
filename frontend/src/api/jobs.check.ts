import assert from "node:assert/strict";
import { setDemoData } from "../lib/demoData";
import { DEFAULT_FILTERS } from "../lib/jobFilters";
import { API_BASE } from "./config";
import {
	fetchAllJobs,
	fetchJob,
	fetchJobOverview,
	fetchJobPage,
	jobPageQuery,
} from "./jobs";

setDemoData(true);
const [first] = await fetchAllJobs();
const detail = await fetchJob(first?.ID ?? "");
assert.ok(detail.Description, "job detail must include full description");

const firstPage = await fetchJobPage({ ...DEFAULT_FILTERS, scored: true }, 25);
assert.equal(firstPage.items.length, 25, "demo page holds one page of jobs");
assert.ok(firstPage.total > 25, "demo total counts every matching job");
const secondPage = await fetchJobPage(
	{ ...DEFAULT_FILTERS, scored: true, page: 2 },
	25,
);
assert.ok(
	secondPage.items.every(
		(job) => !firstPage.items.some((seen) => seen.ID === job.ID),
	),
	"page 2 shares no job with page 1",
);

const newest = await fetchJobPage(
	{ ...DEFAULT_FILTERS, scored: true, sort: "newest" },
	25,
);
const times = newest.items.map((job) =>
	new Date(job.FirstDiscoveredAt ?? job.ScrapedAt).getTime(),
);
assert.deepEqual(
	times,
	[...times].sort((a, b) => b - a),
	"newest sort is descending by discovery time",
);

const searched = await fetchJobPage(
	{ ...DEFAULT_FILTERS, scored: true, q: first?.Title ?? "" },
	25,
);
assert.ok(searched.total >= 1 && searched.total < firstPage.total);
assert.ok(
	searched.items.some((job) => job.ID === first?.ID),
	"search keeps matching jobs",
);
setDemoData(false);

assert.equal(
	jobPageQuery(DEFAULT_FILTERS, 25).toString(),
	"page_size=25&sort=relevant&poster=direct",
);
assert.equal(
	jobPageQuery(
		{
			...DEFAULT_FILTERS,
			page: 3,
			sort: "best",
			q: "go dev",
			company: "c1",
			scored: true,
			suit: 60,
			graded: "ungraded",
			seen: "unseen",
			poster: "agency",
			fav: true,
			src: ["greenhouse", "lever"],
			work: ["remote", "hybrid"],
			sal: true,
			salMin: 70000,
			salMax: 90000,
		},
		25,
	).toString(),
	new URLSearchParams({
		page: "3",
		page_size: "25",
		sort: "best",
		q: "go dev",
		company_id: "c1",
		scored: "1",
		suit: "60",
		graded: "ungraded",
		seen: "unseen",
		poster: "agency",
		favourite: "1",
		source: "greenhouse,lever",
		work: "remote,hybrid",
		salary: "1",
		salary_min: "70000",
		salary_max: "90000",
	}).toString(),
);

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
	assert.deepEqual(requested, [
		`${API_BASE}/jobs?scored=1&page=1&page_size=100`,
		`${API_BASE}/jobs?scored=1&page=2&page_size=100`,
	]);
	assert.deepEqual(
		all.map((job) => job.ID),
		["0", "1", "2"],
		"fetchAllJobs walks every server page",
	);

	requested.length = 0;
	const page = await fetchJobPage({ ...DEFAULT_FILTERS, page: 2 }, 25);
	assert.deepEqual(requested, [
		`${API_BASE}/jobs?${jobPageQuery({ ...DEFAULT_FILTERS, page: 2 }, 25)}`,
	]);
	assert.equal(page.total, 3);
} finally {
	globalThis.fetch = originalFetch;
}

globalThis.fetch = async (input) => {
	requested.push(String(input));
	return Response.json({
		total: 4,
		sources: 2,
		newToday: 1,
		todaySources: 1,
		top: [],
	});
};
try {
	requested.length = 0;
	const overview = await fetchJobOverview(false);
	const url = new URL(requested[0] ?? "");
	assert.equal(requested.length, 1, "overview is one request, not a page walk");
	assert.equal(url.pathname, "/jobs/overview");
	assert.equal(url.searchParams.get("ranked"), "0");
	assert.ok(url.searchParams.get("day_start"), "client sends its day start");
	assert.equal(overview.total, 4);
} finally {
	globalThis.fetch = originalFetch;
}
