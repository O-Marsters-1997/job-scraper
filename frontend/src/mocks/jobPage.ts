import type { JobFilters } from "@/lib/jobFilters";
import type { JobPage } from "@/lib/jobPage";
import type { Job } from "@/types/job";
import { getCompanyKind } from "./companies";
import { isCompanyFavourite } from "./favourites";
import { getGrade, isDismissed } from "./grades";
import { getJobs } from "./jobs";
import { isCompanyExcluded } from "./scoring";
import { isSeen } from "./seen";

const HALF_LIFE_DAYS = 3;
const UNSCORED_PRIOR = 50;
const MS_PER_DAY = 86_400_000;

function parseSalary(
	raw: string | undefined | null,
): { min: number; max: number } | null {
	if (!raw) return null;
	const normalised = raw
		.replace(/[£$€,]/g, "")
		.replace(/\b(\d+(?:\.\d+)?)k\b/gi, (_, n) => String(Number(n) * 1000));
	const [min, second] = [...normalised.matchAll(/\d+(?:\.\d+)?/g)].map((m) =>
		Number(m[0]),
	);
	if (min === undefined) return null;
	return { min, max: second ?? min };
}

function arrangement(job: Job): string {
	const wa = job.WorkArrangement?.toLowerCase().trim();
	if (wa === "remote" || wa === "hybrid" || wa === "onsite") return wa;
	const days = job.DaysInOffice;
	if (days === 0) return "remote";
	if (typeof days === "number" && days >= 5) return "onsite";
	if (typeof days === "number" && days >= 1) return "hybrid";
	return "unknown";
}

function matches(job: Job, f: JobFilters): boolean {
	const q = f.q.toLowerCase();
	if (q) {
		const hay =
			`${job.Title} ${job.CompanySlug} ${job.Location} ${job.Source}`.toLowerCase();
		if (!hay.includes(q)) return false;
	}
	if (f.company && job.CompanyID !== f.company) return false;
	if (f.scored && job.SuitabilityScore == null) return false;
	if (f.suit !== undefined && (job.SuitabilityScore ?? -Infinity) < f.suit)
		return false;
	if (f.graded === "ungraded" && job.Grade) return false;
	if (f.graded === "graded" && !job.Grade) return false;
	if (f.seen === "unseen" && job.Seen) return false;
	if (f.seen === "seen" && !job.Seen) return false;
	if (f.poster !== "all") {
		const agency =
			job.CompanyKind === "agency" || job.CompanyKind === "likely_agency";
		if (agency !== (f.poster === "agency")) return false;
	}
	if (f.fav && !job.CompanyFavourite) return false;
	if (
		f.src.length > 0 &&
		!f.src.some((s) => s.toLowerCase() === job.Source.toLowerCase())
	)
		return false;
	if (f.work.length > 0 && !f.work.includes(arrangement(job))) return false;
	if (f.sal) {
		const parsed = parseSalary(job.SalaryRaw ?? job.SalaryRange);
		if (!parsed) return false;
		if (parsed.max < (f.salMin ?? -Infinity)) return false;
		if (parsed.min > (f.salMax ?? Infinity)) return false;
	}
	return true;
}

const discoveredAt = (job: Job) =>
	new Date(job.FirstDiscoveredAt ?? job.ScrapedAt).getTime() || 0;

function rank(job: Job, now: number): number {
	const ageDays = Math.max(0, (now - discoveredAt(job)) / MS_PER_DAY);
	const score = job.SuitabilityScore ?? UNSCORED_PRIOR;
	return score * 0.5 ** (ageDays / HALF_LIFE_DAYS);
}

function sorted(jobs: Job[], sort: JobFilters["sort"]): Job[] {
	const now = Date.now();
	const byNewest = (a: Job, b: Job) =>
		discoveredAt(b) - discoveredAt(a) || a.ID.localeCompare(b.ID);
	const copy = [...jobs];
	switch (sort) {
		case "newest":
			return copy.sort(byNewest);
		case "best":
			return copy.sort(
				(a, b) =>
					(b.SuitabilityScore ?? -1) - (a.SuitabilityScore ?? -1) ||
					byNewest(a, b),
			);
		case "relevant":
			return copy.sort((a, b) => rank(b, now) - rank(a, now) || byNewest(a, b));
	}
}

export function getJobPage(filters: JobFilters, pageSize: number): JobPage {
	const visible = getJobs()
		.filter(
			(job) => !isDismissed(job.ID) && !isCompanyExcluded(job.CompanySlug),
		)
		.map(
			(job): Job => ({
				...job,
				Grade: getGrade(job.ID)?.grade ?? "",
				Seen: isSeen(job.ID),
				CompanyFavourite: isCompanyFavourite(job.CompanyID),
				CompanyKind: getCompanyKind(job.CompanyID),
			}),
		);
	const matching = sorted(
		visible.filter((job) => matches(job, filters)),
		filters.sort,
	);
	const start = ((filters.page ?? 1) - 1) * pageSize;
	return {
		items: matching.slice(start, start + pageSize),
		total: matching.length,
	};
}
