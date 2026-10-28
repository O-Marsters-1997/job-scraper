import { z } from "zod";

const commentQuoteSchema = z.object({
	exact: z.string(),
	prefix: z.string(),
	suffix: z.string(),
});
export type CommentQuote = z.infer<typeof commentQuoteSchema>;

export const jobCommentSchema = z.object({
	ID: z.string(),
	JobID: z.string(),
	Body: z.string(),
	Quote: commentQuoteSchema,
	CreatedAt: z.string(),
	UpdatedAt: z.string(),
});
export type JobComment = z.infer<typeof jobCommentSchema>;

export const JOB_COMMENT_MAX_LENGTH = 10000;
