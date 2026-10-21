import { z } from "zod";

const trackedDocKindSchema = z.enum(["cv", "letter"]);

export type TrackedDocKind = z.infer<typeof trackedDocKindSchema>;

export const cvSchema = z.object({
	DocID: z.string(),
	TabID: z.string(),
	Title: z.string(),
	SourceDoc: z.string(),
	ModifiedAt: z.string(),
	DocURL: z.string(),
	Visible: z.boolean(),
});

export type CV = z.infer<typeof cvSchema>;
