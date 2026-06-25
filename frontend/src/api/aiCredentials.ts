import { apiFetchVoid } from "./client";

export async function updateAiCredentials(payload: {
	provider: string;
	apiKey: string | null;
}): Promise<void> {
	return apiFetchVoid("/ai-credentials", {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});
}
