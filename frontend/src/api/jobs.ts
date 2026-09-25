import type { Job } from "../types/job";
import { apiFetch } from "./client";
import { mockDelay, useMocks } from "./config";

export interface JobPage {
	items: Job[];
	next_cursor: string;
}

export async function fetchAllJobs(): Promise<Job[]> {
	if (useMocks()) {
		const { getJobs } = await import("../mocks/db");
		await mockDelay();
		return getJobs();
	}
	return apiFetch<Job[]>("/jobs/all");
}

export async function fetchJobs(
	options: {
		cursor?: string | undefined;
		companyId?: string | undefined;
		limit?: number;
	} = {},
): Promise<JobPage> {
	if (useMocks()) {
		const { getCompanies, getJobs } = await import("../mocks/db");
		await mockDelay();
		const company = getCompanies().find(
			(item) => item.ID === options.companyId,
		);
		const jobs = getJobs().filter(
			(job) => !options.companyId || job.CompanySlug === company?.Slug,
		);
		const offset = options.cursor ? Number(options.cursor) : 0;
		const limit = options.limit ?? 100;
		return {
			items: jobs.slice(offset, offset + limit),
			next_cursor: offset + limit < jobs.length ? String(offset + limit) : "",
		};
	}
	const params = new URLSearchParams({ limit: String(options.limit ?? 100) });
	if (options.cursor) params.set("cursor", options.cursor);
	if (options.companyId) params.set("company_id", options.companyId);
	return apiFetch<JobPage>(`/jobs?${params}`);
}

export async function fetchJob(id: string): Promise<Job> {
	if (useMocks()) {
		const { getJobs } = await import("../mocks/db");
		await mockDelay();
		const job = getJobs().find((item) => item.ID === id);
		if (!job) throw new Error("Job not found");
		return job;
	}
	return apiFetch<Job>(`/jobs/${encodeURIComponent(id)}`);
}
