import type { Job } from "../types/job";
import { jobSchema } from "../types/job";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchAllJobs(): Promise<Job[]> {
	return mocked(
		(db) =>
			db
				.getJobs()
				.filter(
					(job) => job.SuitabilityScore != null && !db.isDismissed(job.ID),
				)
				.map((job) => ({
					...job,
					Grade: db.getGrade(job.ID)?.grade ?? "",
					Seen: db.isSeen(job.ID),
				})),
		() => apiFetch("/jobs/all", jobSchema.array()),
	);
}

export async function fetchJob(id: string): Promise<Job> {
	return mocked(
		(db) => {
			const job = db.getJobs().find((item) => item.ID === id);
			if (!job) throw new Error("Job not found");
			return { ...job, Seen: db.isSeen(job.ID) };
		},
		() => apiFetch(`/jobs/${encodeURIComponent(id)}`, jobSchema),
	);
}

export async function markJobsSeen(input: {
	jobIds: string[];
	seen: boolean;
}): Promise<void> {
	return mocked(
		(db) => db.markJobsSeen(input.jobIds, input.seen),
		() =>
			apiFetchVoid(
				"/jobs/seen",
				jsonInit("POST", { JobIDs: input.jobIds, Seen: input.seen }),
			),
	);
}
