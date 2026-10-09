import { z } from "zod";

const sourceFilterOptionSchema = z.object({
	value: z.string(),
	label: z.string(),
});

const sourceFilterFieldSchema = z.object({
	name: z.string(),
	label: z.string(),
	required: z.boolean(),
	options: z.array(sourceFilterOptionSchema).optional(),
});

export type SourceFilterField = z.infer<typeof sourceFilterFieldSchema>;

export const sourceInfoSchema = z.object({
	name: z.string(),
	label: z.string(),
	kind: z.enum(["board", "url", "filter"]),
	role: z.enum(["ats", "discovery"]),
	incremental: z.boolean(),
	url_prefix: z.string(),
	filters: z.array(sourceFilterFieldSchema),
});

export type SourceInfo = z.infer<typeof sourceInfoSchema>;

export const resolvedUrlSchema = z.object({
	kind: z.enum(["search", "ats"]),
	source: z.string(),
	value: z.string(),
	filters: z
		.record(z.string(), z.string())
		.nullish()
		.transform((v) => v ?? {}),
	dropped: z
		.array(z.string())
		.nullish()
		.transform((v) => v ?? []),
	url: z.string(),
});

export type ResolvedURL = z.infer<typeof resolvedUrlSchema>;
