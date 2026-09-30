import assert from "node:assert/strict";
import type { SourceInfo } from "@/types/source";
import type { SourceTarget } from "@/types/sourceTarget";
import {
	describeFilters,
	isDuplicateSearch,
	matchesSearch,
	paginate,
	parseSearchParams,
	sortTargets,
} from "./searchTargets";

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
	URL: "",
	...over,
});

assert.deepEqual(
	parseSearchParams({ tab: "ats", page: "3", sort: "bogus", dir: "desc" }),
	{
		tab: "ats",
		q: undefined,
		src: undefined,
		status: undefined,
		sort: undefined,
		dir: "desc",
		page: 3,
	},
);
assert.deepEqual(parseSearchParams({ page: "1" }).page, undefined);
assert.deepEqual(parseSearchParams({ page: "x" }).page, undefined);

assert.deepEqual(
	describeFilters(
		target({ Filters: { recency: "day", other: "x", empty: "" } }),
		info,
	).map((f) => f.display),
	["Recency: Past 24 hours", "other: x"],
);

const t = target({ Filters: { recency: "day" } });
assert.deepEqual(matchesSearch(t, info, { q: "past 24" }), true);
assert.deepEqual(matchesSearch(t, info, { q: "nope" }), false);
assert.deepEqual(matchesSearch(t, info, { src: "indeed" }), false);
assert.deepEqual(matchesSearch(t, info, { status: "paused" }), false);
assert.deepEqual(
	matchesSearch(target({ RunStatus: "failed" }), info, { status: "failed" }),
	true,
);

const a = target({ ID: "a", Value: "b", LastRunAt: "2026-01-02T00:00:00Z" });
const b = target({ ID: "b", Value: "a", LastRunAt: "2026-01-01T00:00:00Z" });
assert.deepEqual(
	sortTargets([a, b], "search").map((x) => x.ID),
	["b", "a"],
);
assert.deepEqual(
	sortTargets([a, b], "lastRun", "desc").map((x) => x.ID),
	["a", "b"],
);
assert.deepEqual(
	sortTargets([a, b], undefined).map((x) => x.ID),
	["a", "b"],
);

const p = paginate(
	Array.from({ length: 25 }, (_, i) => i),
	9,
);
assert.deepEqual(
	[p.page, p.pageCount, p.from, p.to, p.items.length],
	[3, 3, 21, 25, 5],
);
assert.deepEqual(paginate([], undefined).from, 0);

const existing = [target({ Filters: { recency: "day", empty: "" } })];
assert.deepEqual(
	isDuplicateSearch(existing, {
		source: "linkedin",
		value: "engineer",
		filters: { recency: "day" },
	}),
	true,
);
assert.deepEqual(
	isDuplicateSearch(existing, {
		source: "linkedin",
		value: "engineer",
		filters: {},
	}),
	false,
);
assert.deepEqual(
	isDuplicateSearch(existing, {
		source: "indeed",
		value: "engineer",
		filters: { recency: "day" },
	}),
	false,
);
