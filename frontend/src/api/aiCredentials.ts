import { apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function updateAiCredentials(payload: {
	provider: string;
	apiKey: string | null;
}): Promise<void> {
	return mocked(
		(db) => db.setAiCredential(payload.provider, payload.apiKey),
		() => apiFetchVoid("/ai-credentials", jsonInit("PUT", payload)),
	);
}
