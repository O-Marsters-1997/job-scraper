import { z } from "zod";
import { type DraftFinding, draftFindingSchema } from "@/types/tailoring";

export const promptPartSchema = z.object({
	path: z.string(),
	text: z.string(),
});
export const promptLabSchema = z.object({
	generation: z.string(),
	main: z.string(),
	parts: z.array(promptPartSchema),
	documents: z.array(promptPartSchema),
	checks: z.array(z.string()),
});
export const promptTryDoneSchema = z.object({
	text: z.string(),
	findings: z.array(draftFindingSchema),
	cost: z.number(),
	promptKey: z.string(),
});

export type PromptPart = z.infer<typeof promptPartSchema>;
export type PromptLab = z.infer<typeof promptLabSchema>;
export type PromptTryDone = z.infer<typeof promptTryDoneSchema>;
export type PromptTry = {
	n: number;
	text: string;
	streaming: boolean;
	findings: DraftFinding[];
	lines: number | null;
	changed: string[];
	steer: string;
	cost: number;
};
