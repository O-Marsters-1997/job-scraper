import type { Job } from "@/types/job";

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
	};
}

/** Parse free-text salary strings like "£80,000 to £95,000", "£80k-£95k", "Up to £90k". */
export function parseSalary(
	raw: string | undefined | null,
): { min: number; max: number } | null {
	if (!raw) return null;
	const normalised = raw
		.replace(/[£$€,]/g, "")
		.replace(/\b(\d+(?:\.\d+)?)k\b/gi, (_, n) => String(Number(n) * 1000));
	const nums = [...normalised.matchAll(/\d+(?:\.\d+)?/g)].map((m) =>
		Number(m[0]),
	);
	const [min, second] = nums;
	if (min === undefined) return null;
	const max = second ?? min;
	return { min, max };
}

type Arrangement = "remote" | "hybrid" | "onsite" | "unknown";

/** Derive a canonical work arrangement from the job, covering both live and demo data. */
export function normalizeArrangement(job: Job): Arrangement {
	const wa = job.WorkArrangement?.toLowerCase().trim();
	if (wa === "remote") return "remote";
	if (wa === "hybrid") return "hybrid";
	if (wa === "onsite") return "onsite";
	const d = job.DaysInOffice;
	if (d === 0) return "remote";
	if (typeof d === "number" && d >= 5) return "onsite";
	if (typeof d === "number" && d >= 1) return "hybrid";
	return "unknown";
}

export function applyJobFilters(jobs: Job[], f: JobFilters): Job[] {
	const q = f.q.toLowerCase();
	return jobs.filter((j) => {
		if (q) {
			const hay =
				`${j.Title} ${j.CompanySlug} ${j.Location} ${j.Source}`.toLowerCase();
			if (!hay.includes(q)) return false;
		}
		if (f.company && j.CompanyID !== f.company) return false;
		if (f.scored && j.SuitabilityScore == null) return false;
		if (f.suit !== undefined && (j.SuitabilityScore ?? -Infinity) < f.suit)
			return false;
		if (f.graded === "ungraded" && j.Grade) return false;
		if (f.graded === "graded" && !j.Grade) return false;
		if (f.seen === "unseen" && j.Seen) return false;
		if (f.seen === "seen" && !j.Seen) return false;
		if (f.src.length > 0 && !f.src.includes(j.Source)) return false;
		if (f.work.length > 0 && !f.work.includes(normalizeArrangement(j)))
			return false;
		if (f.sal) {
			const parsed = parseSalary(j.SalaryRaw ?? j.SalaryRange);
			if (!parsed) return false;
			const reqMin = f.salMin ?? -Infinity;
			const reqMax = f.salMax ?? Infinity;
			if (parsed.max < reqMin || parsed.min > reqMax) return false;
		}
		return true;
	});
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
	return n;
}

/** The server-ordered list, where wildcard slots are meaningful: no search, filters or column sort. */
export function isDefaultView(f: JobFilters, sorted: boolean): boolean {
	return !sorted && !f.q && !f.company && activeFilterCount(f) === 0;
}

/** Unique Source values present in the dataset, sorted. */
export function sourceOptions(jobs: Job[]): string[] {
	return [...new Set(jobs.map((j) => j.Source))].sort();
}

export function filterCompanyJobs(
	jobs: Job[],
	company: { ID: string; Slug: string },
): Job[] {
	return jobs.filter((job) =>
		job.CompanyID
			? job.CompanyID === company.ID
			: job.CompanySlug === company.Slug,
	);
}
