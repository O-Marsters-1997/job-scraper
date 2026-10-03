import { z } from "zod";
import { bandSchema, scoreRowSchema } from "./job";

export const scoringStatusSchema = z.object({
	pending: z.number().int(),
});

export type ScoringStatus = z.infer<typeof scoringStatusSchema>;

export const recomputeResultSchema = z.object({
	recomputed: z.number().int(),
});

export type RecomputeResult = z.infer<typeof recomputeResultSchema>;

export const jobScoreSchema = z.object({
	jobId: z.string(),
	score: z.number().int(),
	band: bandSchema.optional(),
	rows: z.array(scoreRowSchema),
});

export type JobScore = z.infer<typeof jobScoreSchema>;

export type CorrectionValue = "yes" | "no";

export type CorrectionTarget = { jobId: string; optionId: string };
