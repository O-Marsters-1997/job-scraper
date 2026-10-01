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

const draftStatusSchema = z.enum(["pending", "running", "ready", "failed"]);

const draftOutcomeSchema = z.enum(["kept", "discarded"]);

const draftFindingSchema = z.object({
	check: z.string(),
	severity: z.enum(["block", "warn", "info"]),
	slotId: z.string().optional(),
	message: z.string(),
	score: z.number().optional(),
});

const provenanceBulletSchema = z.object({
	slotId: z.string(),
	segments: z.array(z.object({ text: z.string(), novel: z.boolean() })),
	achievements: z.array(
		z.object({ id: z.string(), positionId: z.string(), text: z.string() }),
	),
});

const draftProvenanceSchema = z.object({
	positions: z.array(
		z.object({
			positionId: z.string(),
			employer: z.string(),
			title: z.string(),
			bullets: z.array(provenanceBulletSchema),
		}),
	),
});

const draftContentSchema = z.object({
	profile: z.string().nullable(),
	skills: z.array(z.string()),
	positions: z.array(
		z.object({
			positionId: z.string(),
			bullets: z.array(
				z.object({ text: z.string(), achievementIds: z.array(z.string()) }),
			),
		}),
	),
});

export const draftSchema = z.object({
	id: z.string(),
	jobId: z.string(),
	status: draftStatusSchema,
	outcome: draftOutcomeSchema.nullable(),
	draftDocUrl: z.string().nullable(),
	lastError: z.string(),
	createdAt: z.string(),
	findings: z.array(draftFindingSchema),
	provenance: draftProvenanceSchema.nullable(),
	content: draftContentSchema.nullable(),
	base: draftContentSchema.nullable(),
});

export const draftRefSchema = z.object({ id: z.string() });

export type SlotEdit = { slotId: string; text: string };
export type DraftFinding = z.infer<typeof draftFindingSchema>;
export type DraftProvenance = z.infer<typeof draftProvenanceSchema>;
export type DraftContent = z.infer<typeof draftContentSchema>;
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
