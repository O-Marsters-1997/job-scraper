import {
	type ScoringConfig,
	scoringConfigSchema,
} from "../types/scoringConfig";
import { apiFetch } from "./client";
import { useMocks } from "./config";

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
