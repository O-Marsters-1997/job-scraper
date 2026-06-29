import type { Job } from "../types/job";
import type { JobScore } from "../types/jobScore";
import { apiFetch } from "./client";
import { mockDelay, useMocks } from "./config";

export async function fetchJobs(): Promise<Job[]> {
	if (useMocks()) {
		const { getJobs } = await import("../mocks/db");
		await mockDelay();
		return getJobs();
	}
	return apiFetch<Job[]>("/jobs");
}

export async function requestJobReasoning(jobId: string): Promise<JobScore> {
	return apiFetch<JobScore>(`/jobs/${jobId}/reasoning`, { method: "POST" });
}
