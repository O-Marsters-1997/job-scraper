import { apiFetch } from "./client";
import { useMocks } from "./config";

export interface AiPrefs {
	configuredProviders: string[];
	scoringEnabled: boolean;
}

export async function fetchAiPrefs(): Promise<AiPrefs> {
	if (useMocks()) {
		return {
			configuredProviders: ["openrouter"],
			scoringEnabled: true,
		};
	}
	return apiFetch<AiPrefs>("/ai-prefs");
}
