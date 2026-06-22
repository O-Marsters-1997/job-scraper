import type { CV } from "../types/cv";
import { API_BASE } from "./config";

export async function fetchCVTemplates(): Promise<CV[]> {
	const res = await fetch(`${API_BASE}/cv-templates`, {
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to fetch CV templates: ${res.status}`);
	return res.json();
}

export async function addTrackedDoc(url: string): Promise<void> {
	const res = await fetch(`${API_BASE}/tracked-docs`, {
		method: "POST",
		credentials: "include",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ url }),
	});
	if (res.status === 400) {
		const data = await res.json().catch(() => ({}));
		const msg: string = data?.error ?? "";
		throw new Error(
			msg.startsWith("cannot access") ? "access-denied" : "invalid-url",
		);
	}
	if (!res.ok) throw new Error(`Failed to add tracked doc: ${res.status}`);
}

export async function removeTrackedDoc(docId: string): Promise<void> {
	const res = await fetch(`${API_BASE}/tracked-docs/${docId}`, {
		method: "DELETE",
		credentials: "include",
	});
	if (!res.ok) throw new Error(`Failed to remove tracked doc: ${res.status}`);
}

export async function hideTab(docId: string, tabId: string): Promise<void> {
	const res = await fetch(
		`${API_BASE}/tracked-docs/${docId}/tabs/${tabId}/hide`,
		{ method: "POST", credentials: "include" },
	);
	if (!res.ok) throw new Error(`Failed to hide tab: ${res.status}`);
}

export async function showTab(docId: string, tabId: string): Promise<void> {
	const res = await fetch(
		`${API_BASE}/tracked-docs/${docId}/tabs/${tabId}/show`,
		{ method: "POST", credentials: "include" },
	);
	if (!res.ok) throw new Error(`Failed to restore tab: ${res.status}`);
}
