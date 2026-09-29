import { z } from "zod";

export const cvHeadingSchema = z.object({
	text: z.string(),
	positionId: z.string().nullable(),
	confirmed: z.boolean(),
	slotCount: z.number(),
});

export const suggestionSchema = z.object({
	achievementId: z.string(),
	positionId: z.string(),
	text: z.string(),
	score: z.number(),
	preselected: z.boolean(),
});

export const headingMappingSchema = z.object({
	headingText: z.string(),
	positionId: z.string().nullable(),
});

export type CVHeading = z.infer<typeof cvHeadingSchema>;
export type Suggestion = z.infer<typeof suggestionSchema>;
export type HeadingMapping = z.infer<typeof headingMappingSchema>;
