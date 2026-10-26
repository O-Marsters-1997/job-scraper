import { startOfDay } from "date-fns";
import { z } from "zod";
import { buildInsights } from "../lib/insights";
import type { JobFilters } from "../lib/jobFilters";
import type { JobPage } from "../lib/jobPage";
import { highValueJobs, jobStats } from "../lib/overview";
import type { Job, JobInsights, JobOverview } from "../types/job";
import { jobInsightsSchema, jobOverviewSchema, jobSchema } from "../types/job";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

const JOBS_PAGE_SIZE = 100;

export const jobPageSchema = z.object({
	items: z.array(jobSchema),
	total: z.number().nullable(),
});

export function jobPageQuery(
	filters: JobFilters,
	pageSize: number,
): URLSearchParams {
	const query = new URLSearchParams();
	if (filters.page) query.set("page", String(filters.page));
	query.set("page_size", String(pageSize));
	query.set("sort", filters.sort);
	if (filters.q) query.set("q", filters.q);
	if (filters.company) query.set("company_id", filters.company);
	if (filters.scored) query.set("scored", "1");
	if (filters.suit !== undefined) query.set("suit", String(filters.suit));
	if (filters.graded !== "all") query.set("graded", filters.graded);
	if (filters.seen !== "all") query.set("seen", filters.seen);
	if (filters.poster !== "all") query.set("poster", filters.poster);
	if (filters.fav) query.set("favourite", "1");
	if (filters.src.length > 0) query.set("source", filters.src.join(","));
	if (filters.work.length > 0) query.set("work", filters.work.join(","));
	if (filters.sal) query.set("salary", "1");
	if (filters.salMin !== undefined)
		query.set("salary_min", String(filters.salMin));
	if (filters.salMax !== undefined)
		query.set("salary_max", String(filters.salMax));
	return query;
}

export async function fetchJobPage(
	filters: JobFilters,
	pageSize: number,
	knownTotal?: number,
): Promise<JobPage> {
	return mocked(
		(db) => db.getJobPage(filters, pageSize),
		async () => {
			const query = jobPageQuery(filters, pageSize);
			if (knownTotal !== undefined) query.set("skip_total", "1");
			const page = await apiFetch(`/jobs?${query}`, jobPageSchema);
			return { ...page, total: page.total ?? knownTotal ?? page.items.length };
		},
	);
}

export async function fetchAllJobs(): Promise<Job[]> {
	return mocked(
		(db) =>
			db
				.getJobs()
				.filter(
					(job) =>
						job.SuitabilityScore != null &&
						!db.isDismissed(job.ID) &&
						!db.isCompanyExcluded(job.CompanySlug),
				)
				.map((job) => ({
					...job,
					Grade: db.getGrade(job.ID)?.grade ?? "",
					Seen: db.isSeen(job.ID),
					CompanyFavourite: db.isCompanyFavourite(job.CompanyID),
					CompanyKind: db.getCompanyKind(job.CompanyID),
				})),
		async () => {
			const jobs: Job[] = [];
			for (let page = 1; ; page++) {
				const res = await apiFetch(
					`/jobs?scored=1&page=${page}&page_size=${JOBS_PAGE_SIZE}`,
					jobPageSchema,
				);
				jobs.push(...res.items);
				if (res.items.length === 0 || jobs.length >= (res.total ?? 0))
					return jobs;
			}
		},
	);
}

export async function fetchJobOverview(ranked: boolean): Promise<JobOverview> {
	return mocked(
		async () => {
			const jobs = await fetchAllJobs();
			return {
				...jobStats(jobs),
				top: highValueJobs(jobs, ranked).jobs,
			};
		},
		() => {
			const query = new URLSearchParams({
				day_start: startOfDay(new Date()).toISOString(),
				ranked: ranked ? "1" : "0",
			});
			return apiFetch(`/jobs/overview?${query}`, jobOverviewSchema);
		},
	);
}

export async function fetchJobInsights(): Promise<JobInsights> {
	return mocked(
		async () => buildInsights(await fetchAllJobs()),
		() => {
			const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
			return apiFetch(
				`/jobs/insights?${new URLSearchParams({ tz })}`,
				jobInsightsSchema,
			);
		},
	);
}

export async function fetchJob(id: string): Promise<Job> {
	return mocked(
		(db) => {
			const job = db.getJobs().find((item) => item.ID === id);
			if (!job) throw new Error("Job not found");
			return {
				...job,
				Seen: db.isSeen(job.ID),
				CompanyFavourite: db.isCompanyFavourite(job.CompanyID),
				CompanyKind: db.getCompanyKind(job.CompanyID),
			};
		},
		() => apiFetch(`/jobs/${encodeURIComponent(id)}`, jobSchema),
	);
}

const MAX_SEEN_JOB_IDS = 5000;

export async function markJobsSeen(input: {
	jobIds: string[];
	seen: boolean;
}): Promise<void> {
	return mocked(
		(db) => db.markJobsSeen(input.jobIds, input.seen),
		async () => {
			for (let i = 0; i < input.jobIds.length; i += MAX_SEEN_JOB_IDS) {
				await apiFetchVoid(
					"/jobs/seen",
					jsonInit("POST", {
						JobIDs: input.jobIds.slice(i, i + MAX_SEEN_JOB_IDS),
						Seen: input.seen,
					}),
				);
			}
		},
	);
}
