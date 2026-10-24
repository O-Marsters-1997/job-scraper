import { z } from "zod";

const scrapeRunOutcomeSchema = z.enum([
	"running",
	"succeeded",
	"retried",
	"failed",
]);

const scrapeRunSchema = z.object({
	ID: z.string(),
	Kind: z.enum(["search", "board"]),
	Source: z.string(),
	TargetID: z.string(),
	TargetValue: z.string(),
	TargetFilters: z.record(z.string(), z.string()),
	BoardID: z.string(),
	CompanyName: z.string(),
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
	Removed: z.number(),
	Reopened: z.number(),
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

const sourceHealthSchema = z.object({
	Kind: z.enum(["search", "board"]),
	TargetID: z.string(),
	Source: z.string(),
	TargetValue: z.string(),
	TargetFilters: z.record(z.string(), z.string()),
	BoardID: z.string(),
	CompanyName: z.string(),
	Status: z.enum(["red", "amber", "green"]),
	Reason: z.enum([
		"last_failed",
		"failing_streak",
		"retried",
		"no_new",
		"no_runs",
		"ok",
	]),
	Recent: z.array(scrapeRunOutcomeSchema),
	LastRunAt: z.string().nullable(),
	New: z.number(),
	Elsewhere: z.number(),
	Removed: z.number(),
	Retries: z.number(),
});

export const sourceHealthListSchema = z.array(sourceHealthSchema);

export type SourceHealth = z.infer<typeof sourceHealthSchema>;

export const scrapeRunSummarySchema = z.object({
	Total: z.number(),
	New: z.number(),
	Removed: z.number(),
	LastStartedAt: z.string().nullable(),
	Facets: z.record(
		z.enum(["outcome", "kind", "source", "trigger"]),
		z.record(z.string(), z.number()),
	),
	BucketMs: z.number(),
	Histogram: z.array(
		z.object({
			Start: z.string(),
			OK: z.number(),
			Retried: z.number(),
			Failed: z.number(),
		}),
	),
});

export type ScrapeRunSummary = z.infer<typeof scrapeRunSummarySchema>;
export type ScrapeRunFacet = keyof ScrapeRunSummary["Facets"];
