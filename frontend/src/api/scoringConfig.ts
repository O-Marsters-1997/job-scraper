import { API_BASE } from "./config";

export interface ScoringConfig {
	suitabilityRubric: string;
	relevanceCutoff: number;
	notifyThreshold: number;
}

export async function fetchScoringConfig(): Promise<ScoringConfig> {
	const res = await fetch(`${API_BASE}/scoring-config`, {
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to fetch scoring config: ${res.status}`);
	return res.json();
}

export async function updateScoringConfig(
	payload: ScoringConfig,
): Promise<ScoringConfig> {
	const res = await fetch(`${API_BASE}/scoring-config`, {
		method: "PUT",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
	if (!res.ok)
		throw new Error(`Failed to update scoring config: ${res.status}`);
	return res.json();
}
