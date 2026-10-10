import { z } from "zod";

export const runWindowSchema = z.object({
	interval_minutes: z.number().nullable(),
	weekdays: z
		.array(z.number())
		.nullish()
		.transform((v) => v ?? []),
	start: z.string(),
	end: z.string(),
	timezone: z.string(),
});

export type RunWindow = z.infer<typeof runWindowSchema>;

export const sourceTargetSchema = z.object({
	ID: z.string(),
	UserID: z.string(),
	Source: z.string(),
	Value: z.string(),
	Enabled: z.boolean(),
	Filters: z.record(z.string(), z.string()),
	RunStatus: z.enum(["idle", "queued", "running", "succeeded", "failed"]),
	LastRunAt: z.string().nullable(),
	LastRunError: z.string(),
	DisabledReason: z.string(),
	URL: z.string(),
	RunWindow: runWindowSchema,
	NextRunAt: z.string().nullable(),
});

export type SourceTarget = z.infer<typeof sourceTargetSchema>;

export interface CreateSourceTargetPayload {
	source: string;
	value: string;
	enabled?: boolean;
	filters?: Record<string, string>;
	scrape_now?: boolean;
	run_window?: RunWindow | null;
}

export interface UpdateSourceTargetPayload {
	enabled?: boolean;
	check_interval_minutes?: number;
	run_window?: RunWindow | null;
}
