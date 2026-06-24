import { API_BASE } from "./config";

export interface AiPrefs {
	suitabilityModel: string;
	availableModels: string[];
}

export async function fetchAiPrefs(): Promise<AiPrefs> {
	const res = await fetch(`${API_BASE}/ai-prefs`, {
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to fetch AI prefs: ${res.status}`);
	return res.json();
}

export async function updateAiPrefs(
	payload: Pick<AiPrefs, "suitabilityModel">,
): Promise<AiPrefs> {
	const res = await fetch(`${API_BASE}/ai-prefs`, {
		method: "PUT",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
	if (!res.ok) throw new Error(`Failed to update AI prefs: ${res.status}`);
	return res.json();
}
