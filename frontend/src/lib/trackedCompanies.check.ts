import assert from "node:assert/strict";
import type { TrackedCompany } from "@/types/company";
import type { SearchParams } from "./searchTargets";
import { boardCounts, matchesCompany, sortCompanies } from "./trackedCompanies";

const company = (
	name: string,
	over: Partial<TrackedCompany> = {},
): TrackedCompany => ({
	id: name,
	name,
	slug: name.toLowerCase(),
	enabled: true,
	check_interval_minutes: 360,
	boards: [
		{
			id: `${name}-b`,
			source: "greenhouse",
			board_token: name,
			status: "verified",
			url: "",
		},
	],
	open_jobs: 0,
	relevant_jobs: 0,
	last_checked_at: null,
	...over,
});

const acme = company("Acme", { open_jobs: 3, last_checked_at: "2026-01-02" });
const zed = company("Zed", {
	open_jobs: 9,
	last_checked_at: "2026-01-03",
	boards: [
		{
			id: "z",
			source: "lever",
			board_token: "zed",
			status: "candidate",
			url: "",
		},
	],
});
const all = [zed, acme];

const params = (p: SearchParams): SearchParams => p;

assert.deepEqual(matchesCompany(acme, params({ q: " ACM " })), true);
assert.deepEqual(matchesCompany(acme, params({ q: "zed" })), false);
assert.deepEqual(matchesCompany(zed, params({ src: "lever" })), true);
assert.deepEqual(matchesCompany(acme, params({ src: "lever" })), false);

assert.deepEqual(
	sortCompanies(all, "company", "asc").map((c) => c.name),
	["Acme", "Zed"],
);
assert.deepEqual(
	sortCompanies(all, "open", "desc").map((c) => c.name),
	["Zed", "Acme"],
);
assert.deepEqual(
	sortCompanies(all, "checked", "asc").map((c) => c.name),
	["Acme", "Zed"],
);
assert.deepEqual(
	sortCompanies(all, "search", "asc").map((c) => c.name),
	["Zed", "Acme"],
);

assert.deepEqual(
	[...boardCounts(all)],
	[
		["lever", 1],
		["greenhouse", 1],
	],
);
