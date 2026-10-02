import {
	SCORE_FEEDBACK_PAGE_SIZE,
	type ScoreFeedback,
	type ScoreFeedbackKind,
	type ScoreFeedbackPage,
} from "../types/scoreFeedback";

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
	};
	entries = [entry, ...entries];
	return structuredClone(entry);
}
