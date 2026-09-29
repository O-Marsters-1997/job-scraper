import type { SourceInfo } from "@/types/source";
import type { SourceTarget } from "@/types/sourceTarget";
import {
	describeFilters,
	matchesSearch,
	paginate,
	parseSearchParams,
	sortTargets,
} from "./searchTargets";

function eq(a: unknown, b: unknown): void {
	if (JSON.stringify(a) !== JSON.stringify(b))
		throw new Error(
			`expected ${JSON.stringify(a)} to equal ${JSON.stringify(b)}`,
		);
}

const info: SourceInfo = {
	name: "linkedin",
	label: "LinkedIn",
	kind: "filter",
	role: "discovery",
	url_prefix: "",
	filters: [
		{
			name: "recency",
			label: "Recency",
			required: false,
			options: [{ value: "day", label: "Past 24 hours" }],
		},
	],
};

const target = (over: Partial<SourceTarget>): SourceTarget => ({
	ID: "1",
	UserID: "u",
	Source: "linkedin",
	Value: "engineer",
	Enabled: true,
	Filters: {},
	RunStatus: "succeeded",
	LastRunAt: null,
	LastRunError: "",
	...over,
});

eq(parseSearchParams({ tab: "ats", page: "3", sort: "bogus", dir: "desc" }), {
	tab: "ats",
	q: undefined,
	src: undefined,
	status: undefined,
	sort: undefined,
	dir: "desc",
	page: 3,
});
eq(parseSearchParams({ page: "1" }).page, undefined);
eq(parseSearchParams({ page: "x" }).page, undefined);

eq(
	describeFilters(
		target({ Filters: { recency: "day", other: "x", empty: "" } }),
		info,
	).map((f) => f.display),
	["Recency: Past 24 hours", "other: x"],
);

const t = target({ Filters: { recency: "day" } });
eq(matchesSearch(t, info, { q: "past 24" }), true);
eq(matchesSearch(t, info, { q: "nope" }), false);
eq(matchesSearch(t, info, { src: "indeed" }), false);
eq(matchesSearch(t, info, { status: "paused" }), false);
eq(
	matchesSearch(target({ RunStatus: "failed" }), info, { status: "failed" }),
	true,
);

const a = target({ ID: "a", Value: "b", LastRunAt: "2026-01-02T00:00:00Z" });
const b = target({ ID: "b", Value: "a", LastRunAt: "2026-01-01T00:00:00Z" });
eq(
	sortTargets([a, b], "search").map((x) => x.ID),
	["b", "a"],
);
eq(
	sortTargets([a, b], "lastRun", "desc").map((x) => x.ID),
	["a", "b"],
);
eq(
	sortTargets([a, b], undefined).map((x) => x.ID),
	["a", "b"],
);

const p = paginate(
	Array.from({ length: 25 }, (_, i) => i),
	9,
);
eq([p.page, p.pageCount, p.from, p.to, p.items.length], [3, 3, 21, 25, 5]);
eq(paginate([], undefined).from, 0);
