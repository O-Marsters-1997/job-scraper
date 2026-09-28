import { apiFetchVoid } from "./client";
import { mockDelay, mocked } from "./config";

export async function updateAiCredentials(payload: {
	provider: string;
	apiKey: string | null;
}): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.setAiCredential(payload.provider, payload.apiKey);
		},
		() =>
			apiFetchVoid("/ai-credentials", {
				method: "PUT",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify(payload),
			}),
	);
}
