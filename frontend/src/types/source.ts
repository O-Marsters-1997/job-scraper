import { z } from "zod";

export const sourceFilterOptionSchema = z.object({
	value: z.string(),
	label: z.string(),
});

export const sourceFilterFieldSchema = z.object({
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
	url_prefix: z.string(),
	filters: z.array(sourceFilterFieldSchema),
});

export type SourceInfo = z.infer<typeof sourceInfoSchema>;

export const resolvedBoardSchema = z.object({
	source: z.string(),
	value: z.string(),
});

export type ResolvedBoard = z.infer<typeof resolvedBoardSchema>;
