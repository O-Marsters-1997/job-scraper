import type { ScoreFeedback } from "../types/scoreFeedback";

let entries: ScoreFeedback[] = [];

export function listScoreFeedback(): ScoreFeedback[] {
	return structuredClone(entries);
}

export function appendOverallFeedback(reason: string): ScoreFeedback {
	const entry: ScoreFeedback = {
		id: `feedback-${entries.length + 1}`,
		kind: "overall",
		reason,
		model: "typesafe/jev-1.13",
		createdAt: new Date().toISOString(),
	};
	entries = [entry, ...entries];
	return structuredClone(entry);
}
