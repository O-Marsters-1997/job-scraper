import type { FeedbackDirection } from "../../../types/scoreFeedback";

export type HistoryEntry = {
	jobId: string;
	feedbackId: string;
	verdict: FeedbackDirection;
	draft: { overall: string; notes: Record<string, string> };
};

export function pendingIds(ids: string[], skipped: string[]): string[] {
	return [
		...ids.filter((id) => !skipped.includes(id)),
		...skipped.filter((id) => ids.includes(id)),
	];
}
