import {
	type ScoringConfig,
	type ScoringConfigInput,
	scoringConfigSchema,
} from "../types/scoringConfig";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	return mocked(
		(db) => db.getScoringConfig(),
		() => apiFetch("/scoring-config", scoringConfigSchema),
	);
}

export async function updateScoringConfig(
	payload: ScoringConfigInput,
): Promise<ScoringConfig> {
	// Validates 0-100 range client-side before send (HTML min/max can be bypassed)
	const validated = scoringConfigSchema.parse(payload);
	return mocked(
		(db) => db.updateScoringConfig(validated),
		() =>
			apiFetch(
				"/scoring-config",
				scoringConfigSchema,
				jsonInit("PUT", validated),
			),
	);
}
