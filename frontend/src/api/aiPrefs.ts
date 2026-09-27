import { type AiPrefs, aiPrefsSchema } from "../types/aiPrefs";
import { apiFetch } from "./client";
import { useMocks } from "./config";

export async function fetchAiPrefs(): Promise<AiPrefs> {
	if (useMocks()) {
		const { getAiPrefs } = await import("../mocks/db");
		return getAiPrefs();
	}
	return apiFetch("/ai-prefs", undefined, aiPrefsSchema);
}
