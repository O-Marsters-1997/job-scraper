import { z } from "zod";
import { apiFetch } from "./client";

export const scoringConfigSchema = z.object({
	suitabilityRubric: z.string(),
	notifyThreshold: z.number().int().min(0).max(100),
	excludedTitleKeywords: z.array(z.string()),
	excludedCompanies: z.array(z.string()),
	excludedSeniority: z.array(z.string()),
	excludedLocations: z.array(z.string()),
});

export type ScoringConfig = z.infer<typeof scoringConfigSchema>;

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	return apiFetch("/scoring-config", undefined, scoringConfigSchema);
}

export async function updateScoringConfig(
	payload: ScoringConfig,
): Promise<ScoringConfig> {
	// Validates 0-100 range client-side before send (HTML min/max can be bypassed)
	const validated = scoringConfigSchema.parse(payload);
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
