import { z } from "zod";

export const jobNoteSchema = z.object({
	ID: z.string(),
	JobID: z.string(),
	Body: z.string(),
	CreatedAt: z.string(),
	UpdatedAt: z.string(),
});
export type JobNote = z.infer<typeof jobNoteSchema>;

export const JOB_NOTE_MAX_LENGTH = 10000;

export const noteQuoteSchema = z.object({
	exact: z.string(),
	prefix: z.string(),
	suffix: z.string(),
});
export type NoteQuote = z.infer<typeof noteQuoteSchema>;
