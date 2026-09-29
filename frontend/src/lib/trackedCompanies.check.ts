import type { TrackedCompany } from "@/types/company";
import type { SearchParams } from "./searchTargets";
import { boardCounts, matchesCompany, sortCompanies } from "./trackedCompanies";

function eq(a: unknown, b: unknown): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`,
		);
}

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

eq(matchesCompany(acme, params({ q: " ACM " })), true);
eq(matchesCompany(acme, params({ q: "zed" })), false);
eq(matchesCompany(zed, params({ src: "lever" })), true);
eq(matchesCompany(acme, params({ src: "lever" })), false);

eq(
	sortCompanies(all, "company", "asc").map((c) => c.name),
	["Acme", "Zed"],
);
eq(
	sortCompanies(all, "open", "desc").map((c) => c.name),
	["Zed", "Acme"],
);
eq(
	sortCompanies(all, "checked", "asc").map((c) => c.name),
	["Acme", "Zed"],
);
eq(
	sortCompanies(all, "search", "asc").map((c) => c.name),
	["Zed", "Acme"],
);

eq(
	[...boardCounts(all)],
	[
		["lever", 1],
		["greenhouse", 1],
	],
);
