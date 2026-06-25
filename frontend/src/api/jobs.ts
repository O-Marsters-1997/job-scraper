import type { Job } from "../types/job";
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
