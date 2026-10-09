import { z } from "zod";

export const usageLevelSchema = z.enum(["ok", "warn", "critical"]);

export type UsageLevel = z.infer<typeof usageLevelSchema>;

export const quotaSchema = z.object({
	provider: z.string(),
	status: z.enum(["ok", "not_configured", "error"]),
	used: z.number().nullable(),
	limit: z.number().nullable(),
	unit: z.string(),
	percent: z.number().nullable(),
	level: usageLevelSchema,
	resetsAt: z.string().nullable(),
	fetchedAt: z.string().nullable(),
	error: z.string(),
});

export type Quota = z.infer<typeof quotaSchema>;

export const proxyUsageSchema = z.object({
	providers: z.array(quotaSchema),
});

export type ProxyUsage = z.infer<typeof proxyUsageSchema>;
