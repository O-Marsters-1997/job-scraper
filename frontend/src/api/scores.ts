import { recomputeResultSchema, scoringStatusSchema } from "../types/scores";
import { apiFetch } from "./client";
import { useMocks } from "./config";

export async function fetchScoringStatus() {
	if (useMocks()) {
		const { getScoringStatus } = await import("../mocks/db");
		return getScoringStatus();
	}
	return apiFetch("/scores/status", undefined, scoringStatusSchema);
}

export async function recomputeScores() {
	if (useMocks()) {
		const { recomputeScores: mockRecompute } = await import("../mocks/db");
		return mockRecompute();
	}
	return apiFetch(
		"/scores/recompute",
		{ method: "POST" },
		recomputeResultSchema,
	);
}
