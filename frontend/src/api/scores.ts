import { z } from "zod";
import { apiFetch } from "./client";
import { useMocks } from "./config";

const scoringStatusSchema = z.object({
	pending: z.number().int(),
});

const recomputeResultSchema = z.object({
	recomputed: z.number().int(),
});

export type ScoringStatus = z.infer<typeof scoringStatusSchema>;
export type RecomputeResult = z.infer<typeof recomputeResultSchema>;

export async function fetchScoringStatus(): Promise<ScoringStatus> {
	if (useMocks()) {
		const { getScoringStatus } = await import("../mocks/db");
		return getScoringStatus();
	}
	return apiFetch("/scores/status", undefined, scoringStatusSchema);
}

export async function recomputeScores(): Promise<RecomputeResult> {
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
