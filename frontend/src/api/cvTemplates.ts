import type { CV } from "../types/cv";
import { cvSchema } from "../types/cv";
import { apiFetch, apiFetchBlob, apiFetchVoid } from "./client";
import { API_BASE, mockDelay, useMocks } from "./config";

export async function fetchCVTemplates(): Promise<CV[]> {
	if (useMocks()) {
		const { getCVs } = await import("../mocks/db");
		await mockDelay();
		return getCVs();
	}
	return apiFetch("/cv-templates", undefined, cvSchema.array());
}

export async function addTrackedDoc(url: string): Promise<void> {
	if (useMocks()) {
		const { addTrackedDoc: mockAdd } = await import("../mocks/db");
		await mockDelay(80);
		mockAdd(url);
		return;
	}
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
}

export async function removeTrackedDoc(docId: string): Promise<void> {
	if (useMocks()) {
		const { removeTrackedDoc: mockRemove } = await import("../mocks/db");
		await mockDelay(80);
		mockRemove(docId);
		return;
	}
	return apiFetchVoid(`/tracked-docs/${docId}`, { method: "DELETE" });
}

export async function hideTab(docId: string, tabId: string): Promise<void> {
	if (useMocks()) {
		const { setTabVisibility } = await import("../mocks/db");
		await mockDelay(80);
		setTabVisibility(docId, tabId, false);
		return;
	}
	return apiFetchVoid(`/tracked-docs/${docId}/tabs/${tabId}/hide`, {
		method: "POST",
	});
}

export async function showTab(docId: string, tabId: string): Promise<void> {
	if (useMocks()) {
		const { setTabVisibility } = await import("../mocks/db");
		await mockDelay(80);
		setTabVisibility(docId, tabId, true);
		return;
	}
	return apiFetchVoid(`/tracked-docs/${docId}/tabs/${tabId}/show`, {
		method: "POST",
	});
}

export async function fetchCVPdf(
	docId: string,
	tabId: string,
): Promise<ArrayBuffer> {
	if (useMocks()) {
		const { getMockPdfBytes } = await import("../mocks/db");
		await mockDelay();
		return getMockPdfBytes();
	}
	return apiFetchBlob(`/cv-templates/${docId}/${tabId}/pdf`);
}
