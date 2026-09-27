import { z } from "zod";
import { apiFetch } from "./client";
import { useMocks } from "./config";

export const pickSchema = z.object({
	optionId: z.string(),
	stance: z.string(),
	source: z.string(),
});

export const moneySchema = z.object({
	amount: z.number().int().min(0),
	currency: z.string().min(1),
});

export const preferencesSchema = z.object({
	picks: z.array(pickSchema),
	salaryFloor: moneySchema.nullable(),
	blockedTech: z.array(z.string()),
});

export const scoringConfigSchema = z.object({
	preferences: preferencesSchema,
	excludedCompanies: z.array(z.string()),
	excludedLocations: z.array(z.string()),
	notifyThreshold: z.number().int().min(0).max(100),
	updatedAt: z.string(),
});

export type Pick = z.infer<typeof pickSchema>;
export type Money = z.infer<typeof moneySchema>;
export type Preferences = z.infer<typeof preferencesSchema>;
export type ScoringConfig = z.infer<typeof scoringConfigSchema>;

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	if (useMocks()) {
		const { getScoringConfig } = await import("../mocks/db");
		return getScoringConfig();
	}
	return apiFetch("/scoring-config", undefined, scoringConfigSchema);
}

export async function updateScoringConfig(
	payload: ScoringConfig,
): Promise<ScoringConfig> {
	// Validates 0-100 range client-side before send (HTML min/max can be bypassed)
	const validated = scoringConfigSchema.parse(payload);
	if (useMocks()) {
		const { updateScoringConfig: setMockConfig } = await import("../mocks/db");
		return setMockConfig(validated);
	}
	return apiFetch(
		"/scoring-config",
		{
			method: "PUT",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(validated),
		},
		scoringConfigSchema,
	);
}
