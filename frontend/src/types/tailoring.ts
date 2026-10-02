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

const draftStatusSchema = z.enum([
	"pending",
	"running",
	"keeping",
	"ready",
	"failed",
]);

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
	profile: z.object({ slotId: z.string() }).nullable(),
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
	keptAs: z.string(),
	draftDocUrl: z.string().nullable(),
	lastError: z.string(),
	createdAt: z.string(),
	findings: z.array(draftFindingSchema),
	provenance: draftProvenanceSchema.nullable(),
	content: draftContentSchema.nullable(),
	base: draftContentSchema.nullable(),
});

const list = <T extends z.ZodType>(item: T) =>
	z
		.array(item)
		.nullable()
		.transform((v) => v ?? []);

const layoutBorderSchema = z.object({
	width: z.number(),
	color: z.string(),
	padding: z.number(),
	dash: z.string(),
});

const layoutRunSchema = z.object({
	text: z.string(),
	font: z.string(),
	size: z.number(),
	bold: z.boolean(),
	italic: z.boolean(),
	underline: z.boolean(),
	color: z.string(),
	link: z.string(),
});

const layoutBlockSchema = z.object({
	slotId: z.string(),
	section: z.string(),
	align: z.string(),
	lineSpacing: z.number(),
	spaceAbove: z.number(),
	spaceBelow: z.number(),
	indentStart: z.number(),
	indentFirstLine: z.number(),
	borderTop: layoutBorderSchema.nullable(),
	borderBottom: layoutBorderSchema.nullable(),
	tabStops: list(z.object({ offset: z.number(), alignment: z.string() })),
	bullet: z
		.object({ glyph: z.string(), level: z.number(), size: z.number() })
		.nullable(),
	runs: list(layoutRunSchema),
});

export const draftLayoutSchema = z.object({
	page: z.object({
		width: z.number(),
		height: z.number(),
		marginTop: z.number(),
		marginBottom: z.number(),
		marginLeft: z.number(),
		marginRight: z.number(),
	}),
	blocks: list(layoutBlockSchema),
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
export type LayoutBorder = z.infer<typeof layoutBorderSchema>;
export type LayoutRun = z.infer<typeof layoutRunSchema>;
export type LayoutBlock = z.infer<typeof layoutBlockSchema>;
export type DraftLayout = z.infer<typeof draftLayoutSchema>;

const suggestActionSchema = z.enum(["fit", "tighten", "verb", "ask"]);

export const suggestDoneSchema = z.object({
	text: z.string(),
	findings: list(draftFindingSchema),
});

export type SuggestAction = z.infer<typeof suggestActionSchema>;
export type SuggestDone = z.infer<typeof suggestDoneSchema>;
export type SuggestRequest = {
	action: SuggestAction;
	prompt: string;
	text: string;
	maxChars: number;
};
