import { z } from "zod";
import { GRADES } from "./grade";
import { bandSchema, scoreRowSchema } from "./job";

export const scoringPreviewSchema = z.object({
	dismissedInTop5: z.object({
		before: z.number().int(),
		after: z.number().int(),
	}),
	jobs: z.array(
		z.object({
			jobId: z.string(),
			title: z.string(),
			company: z.string(),
			grade: z.enum(GRADES).optional(),
			before: z.number().int(),
			after: z.number().int(),
			afterBand: bandSchema,
			afterRows: z.array(scoreRowSchema),
		}),
	),
});

export type ScoringPreview = z.infer<typeof scoringPreviewSchema>;
