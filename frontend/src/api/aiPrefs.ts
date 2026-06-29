import { apiFetch } from "./client";
import { useMocks } from "./config";

export interface AiPrefs {
	suitabilityModel: string;
	reasoningModel: string;
	availableModels: string[];
	configuredProviders: string[];
	scoringEnabled: boolean;
}

export async function fetchAiPrefs(): Promise<AiPrefs> {
	if (useMocks()) {
		return {
			suitabilityModel: "claude-sonnet-4-6",
			availableModels: ["claude-sonnet-4-6"],
			configuredProviders: ["anthropic"],
			scoringEnabled: true,
		};
	}
	return apiFetch<AiPrefs>("/ai-prefs");
}

export async function updateAiPrefs(
	payload: Pick<AiPrefs, "suitabilityModel" | "reasoningModel">,
): Promise<AiPrefs> {
	return apiFetch<AiPrefs>("/ai-prefs", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
