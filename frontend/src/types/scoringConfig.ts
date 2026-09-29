import { z } from "zod";

export const stanceSchema = z.enum(["nice", "avoid", "block"]);

export type Stance = z.infer<typeof stanceSchema>;

export const pickSchema = z.object({
	optionId: z.string(),
	stance: stanceSchema,
	source: z.string(),
	overridden: z.boolean(),
});

export type Pick = z.infer<typeof pickSchema>;

export const moneySchema = z.object({
	amount: z.number().int().min(0),
	currency: z.string().min(1),
});

export type Money = z.infer<typeof moneySchema>;

export const scoringNumbersSchema = z.object({
	notifyThreshold: z
		.number("Notify score must be a number")
		.int("Notify score must be a whole number")
		.min(0, "Notify score must be between 0 and 100")
		.max(100, "Notify score must be between 0 and 100"),
	salaryFloor: z
		.object({
			amount: z
				.number("Minimum salary must be a number")
				.int("Minimum salary must be a whole number")
				.min(0, "Minimum salary can't be negative"),
			currency: z.string().min(1),
		})
		.nullable(),
});

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
