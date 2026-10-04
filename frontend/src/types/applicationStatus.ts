import { z } from "zod";

export const applicationStatusSchema = z.object({
	ID: z.string(),
	UserID: z.string(),
	Name: z.string(),
	Colour: z.string(),
	ReplyWindowDays: z.number().int().nullable(),
	CreatedAt: z.string(),
});

export type ApplicationStatus = z.infer<typeof applicationStatusSchema>;
