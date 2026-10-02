import { z } from "zod";

export const scoreFeedbackSchema = z.object({
	id: z.string(),
	kind: z.enum(["job", "collection", "overall"]),
	reason: z.string(),
	model: z.string(),
	createdAt: z.string(),
});

export const scoreFeedbackPageSchema = z.object({
	entries: z.array(scoreFeedbackSchema),
	total: z.number(),
});

export type ScoreFeedbackPage = z.infer<typeof scoreFeedbackPageSchema>;

export type ScoreFeedbackKind = ScoreFeedback["kind"];

export const SCORE_FEEDBACK_PAGE_SIZE = 20;

export type ScoreFeedback = z.infer<typeof scoreFeedbackSchema>;

export type OverallFeedbackInput = { reason: string };
