import {
	type OverallFeedbackInput,
	type ScoreFeedback,
	scoreFeedbackSchema,
} from "../types/scoreFeedback";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchScoreFeedback(): Promise<ScoreFeedback[]> {
	return mocked(
		(db) => db.listScoreFeedback(),
		() => apiFetch("/scoring-feedback", scoreFeedbackSchema.array()),
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
