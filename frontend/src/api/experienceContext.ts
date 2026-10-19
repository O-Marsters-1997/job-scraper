import {
	type ContextChange,
	contextChangeSchema,
	type GrillPrompt,
	grillPromptSchema,
} from "../types/experience";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchGrillPrompt(
	tab: { docId: string; tabId: string } | null,
): Promise<GrillPrompt> {
	const query = tab
		? `?${new URLSearchParams({ docId: tab.docId, tabId: tab.tabId })}`
		: "";
	return mocked(
		(db) => db.getGrillPrompt(),
		() => apiFetch(`/experience/grill-prompt${query}`, grillPromptSchema),
	);
}

export async function previewContext(output: string): Promise<ContextChange[]> {
	return mocked(
		(db) => db.previewContext(output),
		() =>
			apiFetch(
				"/experience/context/preview",
				contextChangeSchema.array(),
				jsonInit("POST", { output }),
			),
	);
}

export async function applyContext(changes: ContextChange[]): Promise<void> {
	return mocked(
		(db) => db.applyContext(changes),
		() => apiFetchVoid("/experience/context", jsonInit("POST", { changes })),
	);
}
