import { apiFetch } from "./client";

export interface ScoringConfig {
	suitabilityRubric: string;
	relevanceCutoff: number;
	notifyThreshold: number;
}

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	return apiFetch<ScoringConfig>("/scoring-config");
}

export async function updateScoringConfig(
	payload: ScoringConfig,
): Promise<ScoringConfig> {
	return apiFetch<ScoringConfig>("/scoring-config", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
