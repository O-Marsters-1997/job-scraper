import { recomputeResultSchema, scoringStatusSchema } from "../types/scores";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchScoringStatus() {
	return mocked(
		(db) => db.getScoringStatus(),
		() => apiFetch("/scores/status", undefined, scoringStatusSchema),
	);
}

export async function recomputeScores() {
	return mocked(
		(db) => db.recomputeScores(),
		() =>
			apiFetch("/scores/recompute", { method: "POST" }, recomputeResultSchema),
	);
}
