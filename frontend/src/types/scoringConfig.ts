import { z } from "zod";

export const stanceSchema = z.enum(["nice", "ok", "avoid", "block"]);

export type Stance = z.infer<typeof stanceSchema>;

const pickSchema = z.object({
	optionId: z.string(),
	stance: stanceSchema,
	weight: z.number().int().min(1).max(100).optional(),
	source: z.string(),
	overridden: z.boolean(),
});

export type Pick = z.infer<typeof pickSchema>;

const moneySchema = z.object({
	amount: z.number().int().min(0),
	currency: z.string().min(1),
});

const preferencesSchema = z.object({
	picks: z.array(pickSchema),
	salaryFloor: moneySchema.nullable(),
	preferenceText: z.string(),
});

export const scoringConfigSchema = z.object({
	preferences: preferencesSchema,
	excludedTitleKeywords: z.array(z.string()),
	excludedCompanies: z.array(z.string()),
	excludedLocations: z.array(z.string()),
	requiredLocations: z.array(z.string()),
	requiredTitleKeywords: z.array(z.string()),
	notifyThreshold: z.number().int().min(0).max(100),
	maxJobAgeDays: z.number().int().min(0).max(365),
	updatedAt: z.string(),
	backfillQueued: z.number().int().nonnegative().default(0),
});

export type ScoringConfig = z.infer<typeof scoringConfigSchema>;
export type ScoringConfigInput = z.input<typeof scoringConfigSchema>;
