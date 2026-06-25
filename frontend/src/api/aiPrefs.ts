import { apiFetch } from "./client";

export interface AiPrefs {
	suitabilityModel: string;
	availableModels: string[];
}

export async function fetchAiPrefs(): Promise<AiPrefs> {
	return apiFetch<AiPrefs>("/ai-prefs");
}

export async function updateAiPrefs(
	payload: Pick<AiPrefs, "suitabilityModel">,
): Promise<AiPrefs> {
	return apiFetch<AiPrefs>("/ai-prefs", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
