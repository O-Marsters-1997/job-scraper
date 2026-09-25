import { z } from "zod";
import { apiFetch } from "./client";
import { mockDelay, useMocks } from "./config";

export const scoringCriterionSchema = z.object({
	key: z.string(),
	instructions: z.string(),
	true: z.string(),
	false: z.string(),
	required: z.boolean(),
});

export const scoringQuestionsSchema = z.object({
	profile: z.string(),
	criteria: z.array(scoringCriterionSchema),
	scale: z.array(z.string()),
});

export const scoringConfigSchema = z.object({
	notifyThreshold: z.number().int().min(0).max(100),
	excludedTitleKeywords: z.array(z.string()),
	excludedCompanies: z.array(z.string()),
	excludedSeniority: z.array(z.string()),
	excludedLocations: z.array(z.string()),
	scoringQuestions: scoringQuestionsSchema,
});

export type ScoringCriterion = z.infer<typeof scoringCriterionSchema>;
export type ScoringQuestions = z.infer<typeof scoringQuestionsSchema>;
export type ScoringConfig = z.infer<typeof scoringConfigSchema>;

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	if (useMocks()) {
		await mockDelay();
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
		await mockDelay();
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
