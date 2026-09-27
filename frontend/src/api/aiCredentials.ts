import { apiFetchVoid } from "./client";
import { mockDelay, useMocks } from "./config";

export async function updateAiCredentials(payload: {
	provider: string;
	apiKey: string | null;
}): Promise<void> {
	if (useMocks()) {
		const { setAiCredential } = await import("../mocks/db");
		await mockDelay(80);
		setAiCredential(payload.provider, payload.apiKey);
		return;
	}
	return apiFetchVoid("/ai-credentials", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
