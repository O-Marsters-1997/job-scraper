import { apiFetch, apiFetchVoid } from "./client";
import { MOCK_BUILD } from "./config";

export interface GoogleStatus {
	connected: boolean;
	email?: string;
}

export async function fetchGoogleStatus(): Promise<GoogleStatus> {
	if (MOCK_BUILD) return { connected: false };
	return apiFetch<GoogleStatus>("/google/status");
}

export async function disconnectGoogle(): Promise<void> {
	return apiFetchVoid("/google/link", { method: "DELETE" });
}
