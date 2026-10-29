import {
	type CustomOption,
	type CustomOptionDraft,
	type CustomOptionDraftInput,
	type CustomOptionInput,
	customOptionDraftSchema,
	customOptionSchema,
} from "../types/customOption";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function draftCustomOption(
	input: CustomOptionDraftInput,
): Promise<CustomOptionDraft> {
	return mocked(
		(db) => db.draftCustomOption(input),
		() =>
			apiFetch(
				"/custom-options/draft",
				customOptionDraftSchema,
				jsonInit("POST", input),
			),
	);
}

export async function createCustomOption(
	input: CustomOptionInput,
): Promise<CustomOption> {
	return mocked(
		(db) => db.createCustomOption(input),
		() =>
			apiFetch(
				"/scoring-options/custom",
				customOptionSchema,
				jsonInit("POST", input),
			),
	);
}

export async function updateCustomOption(
	id: string,
	input: CustomOptionInput,
): Promise<CustomOption> {
	return mocked(
		(db) => db.updateCustomOption(id, input),
		() =>
			apiFetch(
				`/scoring-options/custom/${encodeURIComponent(id)}`,
				customOptionSchema,
				jsonInit("PUT", input),
			),
	);
}

export async function deleteCustomOption(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteCustomOption(id),
		() =>
			apiFetchVoid(`/scoring-options/custom/${encodeURIComponent(id)}`, {
				method: "DELETE",
			}),
	);
}
