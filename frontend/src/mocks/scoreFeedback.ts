import {
	type JobFeedbackInput,
	SCORE_FEEDBACK_PAGE_SIZE,
	type ScoreFeedback,
	type ScoreFeedbackKind,
	type ScoreFeedbackPage,
} from "../types/scoreFeedback";
import { getJobs } from "./jobs";

let entries: ScoreFeedback[] = [];
let nextId = 1;

export function listScoreFeedback(
	kind?: ScoreFeedbackKind,
	page = 1,
): ScoreFeedbackPage {
	const matching = entries.filter((e) => !kind || e.kind === kind);
	const start = (page - 1) * SCORE_FEEDBACK_PAGE_SIZE;
	return structuredClone({
		entries: matching.slice(start, start + SCORE_FEEDBACK_PAGE_SIZE),
		total: matching.length,
		currentCount: matching.length,
		outdatedCount: 0,
	});
}

export function deleteScoreFeedback(id: string): void {
	entries = entries.filter((e) => e.id !== id);
}

export function appendOverallFeedback(reason: string): ScoreFeedback {
	const entry: ScoreFeedback = {
		id: `feedback-${nextId++}`,
		kind: "overall",
		reason,
		model: "typesafe/jev-1.13",
		createdAt: new Date().toISOString(),
		picksChanged: false,
		modelChanged: false,
	};
	entries = [entry, ...entries];
	return structuredClone(entry);
}

export function appendJobFeedback(input: JobFeedbackInput): ScoreFeedback {
	const job = getJobs().find((j) => j.ID === input.jobId);
	const entry: ScoreFeedback = {
		id: `feedback-${nextId++}`,
		kind: "job",
		direction: input.direction,
		jobId: input.jobId,
		reason: input.reason,
		model: "typesafe/jev-1.13",
		snapshot: {
			score: job?.SuitabilityScore ?? undefined,
			options: (job?.Breakdown ?? []).map((row) => ({
				optionId: row.key,
				label: row.label,
				question: `Does the role match ${row.label}?`,
				stance: row.stance,
				resolved: row.resolved,
				pYes: row.resolved === "yes" ? 0.9 : 0.05,
				pNo: row.resolved === "no" ? 0.9 : 0.05,
				pNotStated: row.resolved === "unknown" ? 0.9 : 0.05,
				confidence: 0.8,
				known: true,
			})),
		},
		createdAt: new Date().toISOString(),
		picksChanged: false,
		modelChanged: false,
	};
	entries = [entry, ...entries];
	return structuredClone(entry);
}
