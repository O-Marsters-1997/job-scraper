import { z } from "zod";

export const pickSchema = z.object({
	optionId: z.string(),
	stance: z.string(),
	source: z.string(),
	overridden: z.boolean(),
});

export type Pick = z.infer<typeof pickSchema>;

export const moneySchema = z.object({
	amount: z.number().int().min(0),
	currency: z.string().min(1),
});

export type Money = z.infer<typeof moneySchema>;

export const preferencesSchema = z.object({
	picks: z.array(pickSchema),
	salaryFloor: moneySchema.nullable(),
	preferenceText: z.string(),
});

export type Preferences = z.infer<typeof preferencesSchema>;

export const scoringConfigSchema = z.object({
	preferences: preferencesSchema,
	excludedTitleKeywords: z.array(z.string()),
	excludedCompanies: z.array(z.string()),
	excludedLocations: z.array(z.string()),
	notifyThreshold: z.number().int().min(0).max(100),
	updatedAt: z.string(),
	backfillQueued: z.number().int().nonnegative().default(0),
});

export type ScoringConfig = z.infer<typeof scoringConfigSchema>;
export type ScoringConfigInput = z.input<typeof scoringConfigSchema>;
