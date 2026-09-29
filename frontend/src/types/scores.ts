import { z } from "zod";

export const scoringStatusSchema = z.object({
	pending: z.number().int(),
});

export type ScoringStatus = z.infer<typeof scoringStatusSchema>;

export const recomputeResultSchema = z.object({
	recomputed: z.number().int(),
	queued: z.number().int(),
});

export type RecomputeResult = z.infer<typeof recomputeResultSchema>;
