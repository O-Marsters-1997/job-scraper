import { z } from "zod";

export const letterSkeletonSchema = z.object({
	intro: z.string(),
	conclusion: z.string(),
	sections: z.array(z.string()),
});

export type LetterSkeleton = z.infer<typeof letterSkeletonSchema>;
