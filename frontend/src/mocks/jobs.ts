import type { Job, ScoreRow } from "@/types/job";
import type { AddManualJobInput, ManualJobResult } from "../types/manualJob";
import type {
	CorrectionTarget,
	CorrectionValue,
	JobScore,
	RecomputeResult,
} from "../types/scores";
import { slugify } from "./helpers";
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

export function addManualJob(input: AddManualJobInput): ManualJobResult {
	const held = jobs.find((j) => j.URL === input.url);
	if (held) return { job: held, existing: true };
	const now = new Date().toISOString();
	const job: Job = {
		ID: `manual-${jobs.length + 1}`,
		Title: input.title,
		Location: "",
		URL: input.url,
		CompanySlug: slugify(input.company),
		Source: "other",
		PostedAt: null,
		ScrapedAt: now,
		FirstDiscoveredAt: now,
		Description: input.description,
		SuitabilityScore: null,
	};
	jobs = [job, ...jobs];
	return { job, existing: false };
}
