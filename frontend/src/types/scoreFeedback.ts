import { z } from "zod";

export const scoreFeedbackSchema = z.object({
	id: z.string(),
	kind: z.enum(["job", "collection", "overall"]),
	reason: z.string(),
	model: z.string(),
	createdAt: z.string(),
});

export type ScoreFeedback = z.infer<typeof scoreFeedbackSchema>;

export type OverallFeedbackInput = { reason: string };
