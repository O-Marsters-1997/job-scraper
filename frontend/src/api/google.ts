import { API_BASE, MOCK_BUILD } from "./config";

export interface GoogleStatus {
	connected: boolean;
	email?: string;
}

export async function fetchGoogleStatus(): Promise<GoogleStatus> {
	if (MOCK_BUILD) {
		return { connected: false };
	}
	const res = await fetch(`${API_BASE}/google/status`, {
		credentials: "include",
	});
	if (!res.ok) {
		throw new Error(`fetchGoogleStatus: ${res.status}`);
	}
	return res.json() as Promise<GoogleStatus>;
}

export async function disconnectGoogle(): Promise<void> {
	const res = await fetch(`${API_BASE}/google/link`, {
		method: "DELETE",
		credentials: "include",
	});
	if (!res.ok && res.status !== 204) {
		throw new Error(`disconnectGoogle: ${res.status}`);
	}
}
