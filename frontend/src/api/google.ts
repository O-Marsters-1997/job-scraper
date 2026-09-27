import { type GoogleStatus, googleStatusSchema } from "../types/google";
import { apiFetch, apiFetchVoid } from "./client";
import { useMocks } from "./config";

export async function fetchGoogleStatus(): Promise<GoogleStatus> {
	if (useMocks()) {
		const { getGoogleStatus } = await import("../mocks/db");
		return getGoogleStatus();
	}
	return apiFetch("/google/status", undefined, googleStatusSchema);
}

export async function disconnectGoogle(): Promise<void> {
	if (useMocks()) {
		const { disconnectGoogle: mockDisconnect } = await import("../mocks/db");
		mockDisconnect();
		return;
	}
	return apiFetchVoid("/google/link", { method: "DELETE" });
}
