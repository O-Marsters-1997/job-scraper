import type { Job, JobPage } from "../types/job";
import { jobPageSchema, jobSchema } from "../types/job";
import { apiFetch } from "./client";
import { mockDelay, mocked } from "./config";

export async function fetchAllJobs(): Promise<Job[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getJobs();
		},
		() => apiFetch("/jobs/all", undefined, jobSchema.array()),
	);
}

export async function fetchJobs(
	options: {
		cursor?: string | undefined;
		companyId?: string | undefined;
		limit?: number;
	} = {},
): Promise<JobPage> {
	return mocked(
		async (db) => {
			await mockDelay();
			const company = db
				.getCompanies()
				.find((item) => item.ID === options.companyId);
			const jobs = db
				.getJobs()
				.filter(
					(job) => !options.companyId || job.CompanySlug === company?.Slug,
				);
			const offset = options.cursor ? Number(options.cursor) : 0;
			const limit = options.limit ?? 100;
			return {
				items: jobs.slice(offset, offset + limit),
				next_cursor: offset + limit < jobs.length ? String(offset + limit) : "",
			};
		},
		() => {
			const params = new URLSearchParams({
				limit: String(options.limit ?? 100),
			});
			if (options.cursor) params.set("cursor", options.cursor);
			if (options.companyId) params.set("company_id", options.companyId);
			return apiFetch(`/jobs?${params}`, undefined, jobPageSchema);
		},
	);
}

export async function fetchJob(id: string): Promise<Job> {
	return mocked(
		async (db) => {
			await mockDelay();
			const job = db.getJobs().find((item) => item.ID === id);
			if (!job) throw new Error("Job not found");
			return job;
		},
		() => apiFetch(`/jobs/${encodeURIComponent(id)}`, undefined, jobSchema),
	);
}
