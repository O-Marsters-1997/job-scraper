import { z } from "zod";

export const scrapeRunOutcomeSchema = z.enum([
	"running",
	"succeeded",
	"retried",
	"failed",
]);

export const scrapeRunSchema = z.object({
	ID: z.string(),
	Kind: z.enum(["search"]),
	Source: z.string(),
	TargetID: z.string(),
	TargetValue: z.string(),
	TargetFilters: z.record(z.string(), z.string()),
	Trigger: z.enum(["scheduled", "manual"]),
	Outcome: scrapeRunOutcomeSchema,
	StopReason: z.enum(["", "last_page", "page_cap", "failed"]),
	StartedAt: z.string(),
	FinishedAt: z.string().nullable(),
	Pages: z.number(),
	Listings: z.number(),
	New: z.number(),
	Elsewhere: z.number(),
	Repeat: z.number(),
	Deferred: z.number(),
	Retries: z.number(),
	Errors: z.array(
		z.object({
			Page: z.number(),
			Attempt: z.number(),
			Message: z.string(),
			At: z.string(),
		}),
	),
});

export const scrapeRunPageSchema = z.object({
	Items: z.array(scrapeRunSchema),
	NextCursor: z.string().nullable(),
});

export type ScrapeRun = z.infer<typeof scrapeRunSchema>;
export type ScrapeRunOutcome = z.infer<typeof scrapeRunOutcomeSchema>;
export type ScrapeRunPage = z.infer<typeof scrapeRunPageSchema>;
