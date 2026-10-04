import type { Job, ScoreRow } from "@/types/job";
import type {
	CorrectionTarget,
	CorrectionValue,
	JobScore,
	RecomputeResult,
} from "../types/scores";
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

function correctedEffect(stance: string, hit: boolean): ScoreRow["effect"] {
	if (stance === "nice" || stance === "ok") return hit ? "meets" : "misses";
	return hit ? "misses" : "neutral";
}

const uncorrectedRows = new Map<string, ScoreRow>();

export function correctAnswer(
	{ jobId, optionId }: CorrectionTarget,
	value: CorrectionValue | null,
): JobScore {
	const job = jobs.find((j) => j.ID === jobId);
	if (!job || job.SuitabilityScore == null) throw new Error("Job not found");
	const memoKey = `${jobId}|${optionId}`;
	const rows = (job.Breakdown ?? []).map((row): ScoreRow => {
		if (row.key !== optionId) return row;
		if (value === null) return uncorrectedRows.get(memoKey) ?? row;
		if (!row.corrected) uncorrectedRows.set(memoKey, row);
		return {
			...row,
			resolved: value,
			effect: correctedEffect(row.stance, value === "yes"),
			corrected: true,
		};
	});
	jobs = jobs.map((j) => (j.ID === jobId ? { ...j, Breakdown: rows } : j));
	return { jobId, score: job.SuitabilityScore, band: job.Band, rows };
}
