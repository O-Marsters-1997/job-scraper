import {
	type OverallFeedbackInput,
	type ScoreFeedback,
	type ScoreFeedbackKind,
	type ScoreFeedbackPage,
	scoreFeedbackPageSchema,
	scoreFeedbackSchema,
} from "../types/scoreFeedback";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchScoreFeedback(
	kind: ScoreFeedbackKind | undefined,
	page: number,
): Promise<ScoreFeedbackPage> {
	return mocked(
		(db) => db.listScoreFeedback(kind, page),
		() => {
			const params = new URLSearchParams({ page: String(page) });
			if (kind) params.set("kind", kind);
			return apiFetch(`/scoring-feedback?${params}`, scoreFeedbackPageSchema);
		},
	);
}

export async function deleteScoreFeedback(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteScoreFeedback(id),
		() => apiFetchVoid(`/scoring-feedback/${id}`, { method: "DELETE" }),
	);
}

export async function appendOverallFeedback(
	input: OverallFeedbackInput,
): Promise<ScoreFeedback> {
	return mocked(
		(db) => db.appendOverallFeedback(input.reason),
		() =>
			apiFetch(
				"/scoring-feedback/overall",
				scoreFeedbackSchema,
				jsonInit("POST", input),
			),
	);
}
