import type { Job } from "../types/job";
import { API_BASE, mockDelay, useMocks } from "./config";

export async function fetchJobs(): Promise<Job[]> {
	if (useMocks()) {
		const { getJobs } = await import("../mocks/db");
		await mockDelay();
		return getJobs();
	}
	const res = await fetch(`${API_BASE}/jobs`, { credentials: "include" });
	if (!res.ok) throw new Error(`Failed to fetch jobs: ${res.status}`);
	return res.json();
}
