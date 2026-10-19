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
});

export type CoverLetter = z.infer<typeof coverLetterSchema>;

export type ScaffoldInput = {
	jobId: string;
	docId: string;
	tabId: string;
	achievementIds: string[];
};
