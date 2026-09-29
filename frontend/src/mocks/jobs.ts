import type { Job } from "@/types/job";
import type { RecomputeResult } from "../types/scores";
import { seed } from "./seed";

let jobs: Job[] = seed.jobs;

export function getJobs(): Job[] {
	return jobs;
}

export function recomputeScores(): RecomputeResult {
	let recomputed = 0;
	jobs = jobs.map((job) => {
		if (job.SuitabilityScore == null) return job;
		recomputed++;
		return {
			...job,
			SuitabilityScore: Math.min(100, job.SuitabilityScore + 1),
		};
	});
	return { recomputed };
}
