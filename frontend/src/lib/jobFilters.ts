import { POSTER_OPTIONS, type PosterFilter } from "@/types/company";

export const SORT_OPTIONS = ["relevant", "newest", "best"] as const;
export type JobSort = (typeof SORT_OPTIONS)[number];

export const SORT_LABELS: Record<JobSort, string> = {
	relevant: "Relevant",
	newest: "Newest",
	best: "Best match",
};

export const GRADED_OPTIONS = ["all", "ungraded", "graded"] as const;
export type GradedFilter = (typeof GRADED_OPTIONS)[number];

export const SEEN_OPTIONS = ["all", "unseen", "seen"] as const;
export type SeenFilter = (typeof SEEN_OPTIONS)[number];

export interface JobFilters {
	q: string;
	suit?: number | undefined;
	src: string[];
	work: string[];
	sal: boolean;
	salMin?: number | undefined;
	salMax?: number | undefined;
	company?: string | undefined;
	scored: boolean;
	graded: GradedFilter;
	seen: SeenFilter;
	fav: boolean;
	poster: PosterFilter;
	sort: JobSort;
	page?: number | undefined;
}

export const DEFAULT_FILTERS: JobFilters = {
	q: "",
	src: [],
	work: [],
	sal: false,
	suit: undefined,
	salMin: undefined,
	salMax: undefined,
	company: undefined,
	scored: false,
	graded: "all",
	seen: "all",
	fav: false,
	poster: "direct",
	sort: "relevant",
	page: undefined,
};

/** Coerce raw URL search params to JobFilters. Used as the route's validateSearch. */
export function parseSearch(raw: Record<string, unknown>): JobFilters {
	const coerceNum = (v: unknown) => {
		const n = Number(v);
		return Number.isFinite(n) ? n : undefined;
	};
	const coerceArr = (v: unknown): string[] => {
		if (Array.isArray(v)) return v.filter((x) => typeof x === "string");
		if (typeof v === "string" && v) return [v];
		return [];
	};
	const page =
		typeof raw.page === "string" && /^\d+$/.test(raw.page)
			? Number(raw.page)
			: raw.page;
	return {
		q: typeof raw.q === "string" ? raw.q : "",
		suit: coerceNum(raw.suit),
		src: coerceArr(raw.src),
		work: coerceArr(raw.work),
		sal: raw.sal === true || raw.sal === "true",
		salMin: coerceNum(raw.salMin),
		salMax: coerceNum(raw.salMax),
		company:
			typeof raw.company === "string" && raw.company ? raw.company : undefined,
		scored:
			raw.scored === undefined ||
			raw.scored === true ||
			raw.scored === "1" ||
			raw.scored === 1,
		graded: GRADED_OPTIONS.find((o) => o === raw.graded) ?? "all",
		seen: SEEN_OPTIONS.find((o) => o === raw.seen) ?? "all",
		fav: raw.fav === true || raw.fav === "1" || raw.fav === 1,
		poster: POSTER_OPTIONS.find((o) => o === raw.poster) ?? "direct",
		sort: SORT_OPTIONS.find((o) => o === raw.sort) ?? "relevant",
		page:
			typeof page === "number" && Number.isInteger(page) && page > 1
				? page
				: undefined,
	};
}

/** Number of active filters (excluding search, which lives outside the panel). */
export function activeFilterCount(f: JobFilters): number {
	let n = 0;
	if (f.suit !== undefined) n++;
	if (f.src.length > 0) n++;
	if (f.work.length > 0) n++;
	if (f.sal) n++;
	if (f.graded !== "all") n++;
	if (f.seen !== "all") n++;
	if (f.fav) n++;
	return n;
}
