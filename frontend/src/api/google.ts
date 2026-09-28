import { type GoogleStatus, googleStatusSchema } from "../types/google";
import { apiFetch, apiFetchVoid } from "./client";
import { mocked } from "./config";

export async function fetchGoogleStatus(): Promise<GoogleStatus> {
	return mocked(
		(db) => db.getGoogleStatus(),
		() => apiFetch("/google/status", undefined, googleStatusSchema),
	);
}

export async function disconnectGoogle(): Promise<void> {
	return mocked(
		(db) => db.disconnectGoogle(),
		() => apiFetchVoid("/google/link", { method: "DELETE" }),
	);
}
