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

export const draftStatusSchema = z.enum([
	"pending",
	"running",
	"ready",
	"failed",
]);

export const draftSchema = z.object({
	id: z.string(),
	jobId: z.string(),
	status: draftStatusSchema,
	draftDocUrl: z.string().nullable(),
	lastError: z.string(),
});

export const draftRefSchema = z.object({ id: z.string() });

export type DraftStatus = z.infer<typeof draftStatusSchema>;
export type Draft = z.infer<typeof draftSchema>;
export type DraftRef = z.infer<typeof draftRefSchema>;
export type DraftInput = {
	jobId: string;
	docId: string;
	tabId: string;
	achievementIds: string[];
};
export type CVHeading = z.infer<typeof cvHeadingSchema>;
export type Suggestion = z.infer<typeof suggestionSchema>;
export type HeadingMapping = z.infer<typeof headingMappingSchema>;
