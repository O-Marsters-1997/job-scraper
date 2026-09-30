import type { Job } from "../types/job";
import { jobSchema } from "../types/job";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchAllJobs(): Promise<Job[]> {
	return mocked(
		(db) => db.getJobs(),
		() => apiFetch("/jobs/all", jobSchema.array()),
	);
}

export async function fetchJob(id: string): Promise<Job> {
	return mocked(
		(db) => {
			const job = db.getJobs().find((item) => item.ID === id);
			if (!job) throw new Error("Job not found");
			return job;
		},
		() => apiFetch(`/jobs/${encodeURIComponent(id)}`, jobSchema),
	);
}
