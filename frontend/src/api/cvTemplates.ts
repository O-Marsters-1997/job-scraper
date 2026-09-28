import type { CV } from "../types/cv";
import { cvSchema } from "../types/cv";
import { apiFetch, apiFetchBlob, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, mocked } from "./config";

export async function fetchCVTemplates(): Promise<CV[]> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getCVs();
		},
		() => apiFetch("/cv-templates", undefined, cvSchema.array()),
	);
}

export async function addTrackedDoc(url: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.addTrackedDoc(url);
		},
		async () => {
			const response = await fetch(`${API_BASE}/tracked-docs`, {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ url }),
			});
			if (response.status === 400) {
				const data = await response.json().catch(() => ({}));
				const msg: string = data?.error ?? "";
				throw new Error(
					msg.startsWith("cannot access") ? "access-denied" : "invalid-url",
				);
			}
			if (!response.ok)
				throw new Error(`Failed to add tracked doc: ${response.status}`);
		},
	);
}

export async function removeTrackedDoc(docId: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.removeTrackedDoc(docId);
		},
		() => apiFetchVoid(`/tracked-docs/${docId}`, { method: "DELETE" }),
	);
}

export async function hideTab(docId: string, tabId: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.setTabVisibility(docId, tabId, false);
		},
		() =>
			apiFetchVoid(`/tracked-docs/${docId}/tabs/${tabId}/hide`, {
				method: "POST",
			}),
	);
}

export async function showTab(docId: string, tabId: string): Promise<void> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			db.setTabVisibility(docId, tabId, true);
		},
		() =>
			apiFetchVoid(`/tracked-docs/${docId}/tabs/${tabId}/show`, {
				method: "POST",
			}),
	);
}

export async function fetchCVPdf(
	docId: string,
	tabId: string,
): Promise<ArrayBuffer> {
	return mocked(
		async (db) => {
			await mockDelay();
			return db.getMockPdfBytes();
		},
		() => apiFetchBlob(`/cv-templates/${docId}/${tabId}/pdf`),
	);
}
