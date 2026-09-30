import type { CV } from "../types/cv";
import { cvSchema } from "../types/cv";
import {
	ApiError,
	apiFetch,
	apiFetchBlob,
	apiFetchVoid,
	jsonInit,
} from "./client";
import { mocked } from "./config";

export async function fetchCVTemplates(): Promise<CV[]> {
	return mocked(
		(db) => db.getCVs(),
		() => apiFetch("/cv-templates", cvSchema.array()),
	);
}

export async function addTrackedDoc(url: string): Promise<void> {
	return mocked(
		(db) => db.addTrackedDoc(url),
		() =>
			apiFetchVoid("/tracked-docs", jsonInit("POST", { url })).catch((err) => {
				if (err instanceof ApiError && err.status === 400) {
					const msg = (err.body as { error?: string } | undefined)?.error ?? "";
					throw new Error(
						msg.startsWith("cannot access") ? "access-denied" : "invalid-url",
					);
				}
				throw err;
			}),
	);
}

export async function hideTab(docId: string, tabId: string): Promise<void> {
	return mocked(
		(db) => db.setTabVisibility(docId, tabId, false),
		() =>
			apiFetchVoid(`/tracked-docs/${docId}/tabs/${tabId}/hide`, {
				method: "POST",
			}),
	);
}

export async function showTab(docId: string, tabId: string): Promise<void> {
	return mocked(
		(db) => db.setTabVisibility(docId, tabId, true),
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
		(db) => db.getMockPdfBytes(),
		() => apiFetchBlob(`/cv-templates/${docId}/${tabId}/pdf`),
	);
}
