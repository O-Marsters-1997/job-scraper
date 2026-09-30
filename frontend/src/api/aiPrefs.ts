import { type AiPrefs, aiPrefsSchema } from "../types/aiPrefs";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchAiPrefs(): Promise<AiPrefs> {
	return mocked(
		(db) => db.getAiPrefs(),
		() => apiFetch("/ai-prefs", aiPrefsSchema),
	);
}
