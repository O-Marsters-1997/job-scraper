import type { ScrapeRunFilters } from "@/api/scrapeRuns";
import type { ScrapeRunFacet } from "@/types/scrapeRun";

export const RANGE_PRESETS = {
	"1h": "Last hour",
	"6h": "Last 6 hours",
	"24h": "Last 24 hours",
	"7d": "Last 7 days",
	"30d": "Last 30 days",
} as const;

export const DEFAULT_RANGE = "7d";
export const PAGE_SIZE = 50;

export interface ActivitySearch extends ScrapeRunFilters {
	before?: string | undefined;
	trail?: string | undefined;
}

export function parseActivitySearch(
	raw: Record<string, unknown>,
): ActivitySearch {
	const text = (v: unknown) => (typeof v === "string" && v ? v : undefined);
	const before = text(raw.before);
	const range = text(raw.range);
	const custom = text(raw.from) && text(raw.to);
	return {
		target: text(raw.target),
		q: text(raw.q),
		outcome: text(raw.outcome),
		kind: text(raw.kind),
		source: text(raw.source),
		trigger: text(raw.trigger),
		range: !custom && range && range in RANGE_PRESETS ? range : undefined,
		from: custom ? text(raw.from) : undefined,
		to: custom ? text(raw.to) : undefined,
		before,
		trail: before ? text(raw.trail) : undefined,
	};
}

export function runFilters(search: ActivitySearch): ScrapeRunFilters {
	const { before: _before, trail: _trail, ...filters } = search;
	return filters;
}

export function facetValues(search: ActivitySearch, facet: ScrapeRunFacet) {
	return search[facet]?.split(",") ?? [];
}

function withFilter(
	search: ActivitySearch,
	patch: Partial<ActivitySearch>,
): ActivitySearch {
	return { ...search, ...patch, before: undefined, trail: undefined };
}

function joined(values: string[]) {
	return values.length > 0 ? values.join(",") : undefined;
}

export function toggleFacetValue(
	search: ActivitySearch,
	facet: ScrapeRunFacet,
	value: string,
): ActivitySearch {
	const current = facetValues(search, facet);
	const next = current.includes(value)
		? current.filter((v) => v !== value)
		: [...current, value];
	return withFilter(search, { [facet]: joined(next) });
}

export function onlyFacetValue(
	search: ActivitySearch,
	facet: ScrapeRunFacet,
	value: string,
): ActivitySearch {
	return withFilter(search, { [facet]: value });
}

export function resetFacet(
	search: ActivitySearch,
	facet: ScrapeRunFacet,
): ActivitySearch {
	return withFilter(search, { [facet]: undefined });
}

export function setQuery(search: ActivitySearch, q: string): ActivitySearch {
	return withFilter(search, { q: q.trim() || undefined });
}

export function setRange(search: ActivitySearch, range: string) {
	return withFilter(search, {
		range: range === DEFAULT_RANGE ? undefined : range,
		from: undefined,
		to: undefined,
	});
}

export function setCustomRange(
	search: ActivitySearch,
	from: string,
	to: string,
) {
	return withFilter(search, { range: undefined, from, to });
}

const FACETS: ScrapeRunFacet[] = ["outcome", "kind", "source", "trigger"];

export function activeFilterCount(search: ActivitySearch): number {
	return FACETS.reduce((n, f) => n + facetValues(search, f).length, 0);
}

export function clearFilters(search: ActivitySearch): ActivitySearch {
	return withFilter(search, {
		outcome: undefined,
		kind: undefined,
		source: undefined,
		trigger: undefined,
		q: undefined,
	});
}

export const MAX_CUSTOM_RANGE_MS = 30 * 24 * 60 * 60 * 1000;

export function customRangeError(from: string, to: string): string | null {
	const start = Date.parse(from);
	const end = Date.parse(to);
	if (Number.isNaN(start) || Number.isNaN(end)) return "Pick both dates";
	if (end <= start) return "To must be after From";
	if (end - start > MAX_CUSTOM_RANGE_MS) return "At most 30 days apart";
	return null;
}

export function pageSummary(
	search: ActivitySearch,
	shown: number,
	total: number,
): string {
	if (shown === 0) return `0 of ${total.toLocaleString()}`;
	const pageIndex =
		(search.before ? 1 : 0) + (search.trail?.split(",").length ?? 0);
	const first = pageIndex * PAGE_SIZE + 1;
	return `${first.toLocaleString()}–${(first + shown - 1).toLocaleString()} of ${total.toLocaleString()}`;
}

export function olderPage(
	search: ActivitySearch,
	next: string,
): ActivitySearch {
	const trail = [search.trail, search.before].filter(Boolean).join(",");
	return { ...search, before: next, trail: trail || undefined };
}

export function newerPage(search: ActivitySearch): ActivitySearch {
	const trail = search.trail?.split(",") ?? [];
	const before = trail.pop();
	return { ...search, before, trail: trail.join(",") || undefined };
}

export function formatDuration(startedAt: string, finishedAt: string): string {
	const seconds = Math.round(
		(Date.parse(finishedAt) - Date.parse(startedAt)) / 1000,
	);
	if (seconds < 60) return `${seconds}s`;
	return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}
