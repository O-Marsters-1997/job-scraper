import {
	type ScoringConfig,
	type ScoringConfigInput,
	scoringConfigSchema,
} from "../types/scoringConfig";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	return mocked(
		(db) => db.getScoringConfig(),
		() => apiFetch("/scoring-config", undefined, scoringConfigSchema),
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
				{
					method: "PUT",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(validated),
				},
				scoringConfigSchema,
			),
	);
}
