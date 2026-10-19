import { z } from "zod";

export const letterSkeletonSchema = z.object({
	intro: z.string(),
	conclusion: z.string(),
	sections: z.array(z.string()),
});

export type LetterSkeleton = z.infer<typeof letterSkeletonSchema>;

export const coverLetterSchema = z.object({
	id: z.string(),
	jobId: z.string(),
	sourceDocId: z.string(),
	sourceTabId: z.string(),
	tabUrl: z.string(),
	achievementIds: z.array(z.string()),
	cost: z.number(),
	createdAt: z.string(),
	doneAt: z.string().nullable(),
});

export type CoverLetter = z.infer<typeof coverLetterSchema>;

export const letterFindingSchema = z.object({
	kind: z.enum([
		"leftover_bullet",
		"leftover_gap",
		"length",
		"ai_tell",
		"restates_achievement",
		"missing_company",
		"missing_role",
	]),
	text: z.string(),
	achievementId: z.string().optional(),
});

export const letterCheckSchema = z.object({
	words: z.number(),
	findings: z.array(letterFindingSchema),
});

export type LetterFinding = z.infer<typeof letterFindingSchema>;
export type LetterCheck = z.infer<typeof letterCheckSchema>;

export type ScaffoldInput = {
	jobId: string;
	docId: string;
	tabId: string;
	achievementIds: string[];
};
