import { z } from "zod";

const feedbackOptionSchema = z.object({
	optionId: z.string(),
	label: z.string(),
	question: z.string(),
	stance: z.string(),
	resolved: z.string(),
	pYes: z.number(),
	pNo: z.number(),
	pNotStated: z.number(),
	confidence: z.number(),
	known: z.boolean(),
});

export type FeedbackOption = z.infer<typeof feedbackOptionSchema>;

const snapshotSchema = z.object({
	score: z.number().optional(),
	options: z.array(feedbackOptionSchema).optional(),
});

export const scoreFeedbackSchema = z.object({
	id: z.string(),
	kind: z.enum(["job", "collection", "overall"]),
	direction: z.enum(["higher", "lower"]).optional(),
	jobId: z.string().optional(),
	reason: z.string(),
	model: z.string(),
	snapshot: snapshotSchema.optional(),
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

export type FeedbackDirection = NonNullable<ScoreFeedback["direction"]>;

export type JobFeedbackInput = {
	jobId: string;
	direction: FeedbackDirection;
	reason: string;
};
