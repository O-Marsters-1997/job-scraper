import type { SourceInfo } from "@/types/source";
import type { SourceTarget } from "@/types/sourceTarget";

export const SEARCH_PAGE_SIZE = 10;

export type SearchTab = "boards" | "ats";
export type SearchSortKey =
	| "search"
	| "lastRun"
	| "company"
	| "open"
	| "checked";
export type SortDir = "asc" | "desc";
export type SearchStatus = "active" | "paused" | "failed";

export interface SearchParams {
	tab?: "ats" | undefined;
	q?: string | undefined;
	src?: string | undefined;
	status?: SearchStatus | undefined;
	sort?: SearchSortKey | undefined;
	dir?: SortDir | undefined;
	page?: number | undefined;
}

export function parseSearchParams(raw: Record<string, unknown>): SearchParams {
	const page = Number(raw.page);
	return {
		tab: raw.tab === "ats" ? "ats" : undefined,
		q: typeof raw.q === "string" && raw.q ? raw.q : undefined,
		src: typeof raw.src === "string" && raw.src ? raw.src : undefined,
		status:
			raw.status === "active" ||
			raw.status === "paused" ||
			raw.status === "failed"
				? raw.status
				: undefined,
		sort:
			raw.sort === "search" ||
			raw.sort === "lastRun" ||
			raw.sort === "company" ||
			raw.sort === "open" ||
			raw.sort === "checked"
				? raw.sort
				: undefined,
		dir: raw.dir === "asc" || raw.dir === "desc" ? raw.dir : undefined,
		page: Number.isInteger(page) && page > 1 ? page : undefined,
	};
}

export interface FilterChip {
	key: string;
	label: string;
	display: string;
}

export function describeFilters(
	target: SourceTarget,
	info: SourceInfo | undefined,
): FilterChip[] {
	return Object.entries(target.Filters)
		.filter(([, value]) => value !== "")
		.map(([key, value]) => {
			const field = info?.filters.find((f) => f.name === key);
			const optionLabel = field?.options?.find((o) => o.value === value)?.label;
			const label = field?.label ?? key;
			return { key, label, display: `${label}: ${optionLabel ?? value}` };
		});
}

export function matchesSearch(
	target: SourceTarget,
	info: SourceInfo | undefined,
	params: SearchParams,
): boolean {
	const needle = params.q?.trim().toLowerCase();
	if (needle) {
		const haystack = [
			target.Value,
			...describeFilters(target, info).map((f) => f.display),
		]
			.join(" ")
			.toLowerCase();
		if (!haystack.includes(needle)) return false;
	}
	if (params.src && target.Source !== params.src) return false;
	switch (params.status) {
		case "active":
			return target.Enabled;
		case "paused":
			return !target.Enabled;
		case "failed":
			return target.RunStatus === "failed";
		default:
			return true;
	}
}

export function sortTargets(
	targets: SourceTarget[],
	sort: SearchSortKey | undefined,
	dir: SortDir = "asc",
): SourceTarget[] {
	if (sort !== "search" && sort !== "lastRun") return targets;
	const by =
		sort === "search"
			? (a: SourceTarget, b: SourceTarget) => a.Value.localeCompare(b.Value)
			: (a: SourceTarget, b: SourceTarget) =>
					new Date(a.LastRunAt ?? 0).getTime() -
					new Date(b.LastRunAt ?? 0).getTime();
	const sign = dir === "asc" ? 1 : -1;
	return [...targets].sort((a, b) => sign * by(a, b));
}

export function paginate<T>(
	items: T[],
	page: number | undefined,
	pageSize = SEARCH_PAGE_SIZE,
) {
	const pageCount = Math.max(1, Math.ceil(items.length / pageSize));
	const current = Math.min(page ?? 1, pageCount);
	const start = (current - 1) * pageSize;
	return {
		page: current,
		pageCount,
		total: items.length,
		items: items.slice(start, start + pageSize),
		from: items.length ? start + 1 : 0,
		to: Math.min(start + pageSize, items.length),
	};
}

export function relativeTime(iso: string | null, now = Date.now()): string {
	if (!iso) return "Never";
	const mins = Math.round((now - new Date(iso).getTime()) / 60000);
	if (mins < 1) return "Just now";
	if (mins < 60) return `${mins}m ago`;
	const hours = Math.round(mins / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.round(hours / 24)}d ago`;
}
