import { z } from "zod";
import { apiFetch } from "./client";

const scoringStatusSchema = z.object({
	pending: z.number().int(),
	failed: z.number().int(),
	stale: z.number().int(),
});

export function fetchScoringStatus() {
	return apiFetch("/scores/status", undefined, scoringStatusSchema);
}

export function queueRescore() {
	return apiFetch(
		"/scores/rescore",
		{ method: "POST" },
		z.object({ queued: z.number().int() }),
	);
}
