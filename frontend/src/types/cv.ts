import { z } from "zod";

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
