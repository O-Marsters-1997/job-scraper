import { type Quota, quotaSchema } from "../types/quota";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchAiUsage(): Promise<Quota> {
	return mocked(
		(db) => db.getAiUsage(),
		() => apiFetch("/ai-usage", quotaSchema),
	);
}
