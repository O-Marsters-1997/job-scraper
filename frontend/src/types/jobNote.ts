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
